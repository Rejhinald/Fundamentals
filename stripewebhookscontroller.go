package controllers

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"io/ioutil"
	"net/http"

	"grooper/app/constants"
	"grooper/app/mail"
	"grooper/app/models"
	ops "grooper/app/operations"
	"grooper/app/utils"

	"github.com/revel/modules/jobs/app/jobs"
	"github.com/revel/revel"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/customer"
	"github.com/stripe/stripe-go/v72/invoice"
	"github.com/stripe/stripe-go/v72/paymentintent"
	"github.com/stripe/stripe-go/v72/paymentmethod"
	"github.com/stripe/stripe-go/v72/sub"
	"github.com/stripe/stripe-go/v72/webhook"
)

type StripeWebHooksz struct {
	*revel.Controller
}

func init() {
	revel.FilterController(StripeWebHooksz{}).Insert(StripeFilter, revel.BEFORE, revel.ParamsFilter)
}

var StripeFilter = func(c *revel.Controller, fc []revel.Filter) {

	realRequest := c.Request.In.GetRaw().(*http.Request)
	buf, _ := ioutil.ReadAll(realRequest.Body)
	rdr := ioutil.NopCloser(bytes.NewBuffer(buf))

	c.Args["body"] = buf

	realRequest.Body = rdr

	fc[0](c, fc[1:])
}

func (c StripeWebHooksz) HandleWebhooks() revel.Result {

	buf := c.Args["body"].([]byte)

	// secrets
	endpointSecret, _ := revel.Config.String("stripewebkeys")
	signedSignature := c.Request.Header.Get("Stripe-Signature")

	//
	//

	// run signature verification and parameters
	event, err := webhook.ConstructEvent(buf, signedSignature, endpointSecret)
	if err != nil {
		// revel.AppLog.Errorf("Error running signature verifications. ERROR: %v", err)
		c.Response.Status = http.StatusBadRequest
		return c.RenderJSON("")
	}

	frontendUrl, _ := revel.Config.String("url.frontend")

	result := make(map[string]interface{})

	//handle events here
	switch event.Type {

	case "customer.subscription.created":
		// Extract data from the response
		var customerSubscription stripe.Subscription
		var prevSubscription stripe.Subscription
		// var eventObject stripe.EventData

		//Get Previous Attribute
		jsonbody, err := json.Marshal(event.Data.PreviousAttributes)
		if err != nil {
			// revel.AppLog.Errorf("Cannot parse webhook JSON PreviousAttributes. EVENT: %v", event.Type)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "94"
			return c.RenderJSON(result)
		}

		// unmarshal new subscription values
		err = json.Unmarshal(event.Data.Raw, &customerSubscription)
		if err != nil {
			// revel.AppLog.Errorf("Cannot parse webhook JSON customerSubscription. EVENT: %v", event.Type)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "105"
			return c.RenderJSON(result)
		}

		// unmarshal previous subscription values
		err = json.Unmarshal(jsonbody, &prevSubscription)
		if err != nil {
			// revel.AppLog.Errorf("Cannot unmarshal previous attributes. EVENT: %v", event.Type)
			c.Response.Status = http.StatusBadRequest
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "117"
			return c.RenderJSON(result)
		}

		cus, _ := customer.Get(customerSubscription.Customer.ID, nil)

		//
		//
		//

		// get user by id
		u, opsError := ops.GetUserByEmail(cus.Email)
		if opsError != nil {
			// revel.AppLog.Errorf("Cannot get user information. EVENT: %v", event.Type)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "130"
			return c.RenderJSON(result)
		}

		//subscription data shorthand
		subscriptionData := customerSubscription.Items.Data[0]
		// prevSubscriptionData := prevSubscription.Items.Data[0]

		curSubs, err := ops.GetSubscriptionInfo(u.ActiveCompany, u.UserID)
		if err != nil {
			// revel.AppLog.Errorf("Fail to get current subscription. EVENT: %v", event.Type)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "144"
			return c.RenderJSON(result)
		}

		if curSubs.StripeSubscriptionID == nil {
			uid := utils.GenerateTimestampWithUID()

			// set update parameters
			params := models.Subscription{
				PK:                   utils.AppendPrefix(constants.PREFIX_SUBSCRIPTION, uid),
				SK:                   utils.AppendPrefix(constants.PREFIX_COMPANY, u.ActiveCompany),
				SubscriptionID:       uid,
				CompanyID:            u.ActiveCompany,
				UserID:               u.UserID,
				NoOfUsers:            strconv.FormatInt(subscriptionData.Quantity, 10),
				BillingType:          string(subscriptionData.Price.ID),
				Price:                strconv.FormatInt(subscriptionData.Price.UnitAmount, 10),
				Status:               strings.ToUpper(string(customerSubscription.Status)),
				EndDate:              strconv.FormatInt(customerSubscription.CurrentPeriodEnd, 10),
				StripeSubscriptionID: &subscriptionData.Subscription,
				SentTrialWarning:     "FALSE", //Remove this if we will use customer.subscription.trial_will_end event
				CreatedAt:            utils.GetCurrentTimestamp(),
				Type:                 constants.ENTITY_TYPE_SUBSCRIPTION,
				IsUpdated:            constants.BOOL_TRUE,
			}

			// Add subscription
			_, addSubscriptionError := ops.AddSubscription(params)
			if addSubscriptionError != nil {
				// revel.AppLog.Errorf("Fail to add subscription. EVENT: %v", event.Type)
				c.Response.Status = http.StatusBadRequest
				result["error"] = err
				result["event"] = event.Type
				result["line"] = "177"
				return c.RenderJSON(result)
			}
		} else {
			if curSubs.Status == constants.ITEM_STATUS_CANCELED {
				var Recipients []mail.Recipient

				Recipient := mail.Recipient{
					Email:        cus.Email,
					RedirectLink: frontendUrl + "/dashboard",
				}

				Recipients = append(Recipients, Recipient)

				jobs.Now(mail.SendEmail{
					Subject:    "Welcome back!",
					Template:   "welcome-back.html",
					Recipients: Recipients,
				})
			}

			params := models.Subscription{
				CompanyID:            curSubs.CompanyID,
				SubscriptionID:       curSubs.SubscriptionID,
				NoOfUsers:            strconv.FormatInt(subscriptionData.Quantity, 10),
				BillingType:          string(subscriptionData.Price.ID),
				Price:                strconv.FormatInt(subscriptionData.Price.UnitAmount, 10),
				Status:               strings.ToUpper(string(customerSubscription.Status)),
				EndDate:              strconv.FormatInt(customerSubscription.CurrentPeriodEnd, 10),
				StripeSubscriptionID: &subscriptionData.Subscription,
				CancelAtPeriodEnd:    strings.ToUpper(strconv.FormatBool(customerSubscription.CancelAtPeriodEnd)),
				FromTrial:            constants.BOOL_FALSE,
				SentTrialWarning:     "FALSE", //Remove this if we will use customer.subscription.trial_will_end event
				IsUpdated:            constants.BOOL_TRUE,
			}

			updateSubscriptionError := ops.UpdateLocalSubscription(params)
			if updateSubscriptionError != nil {

				//
				// revel.AppLog.Errorf("Fail to update subscription. EVENT: %v SUB-ID: %v", event.Type, curSubs.SubscriptionID)
				c.Response.Status = http.StatusBadRequest
				result["error"] = err
				result["event"] = event.Type
				result["line"] = "221"
				return c.RenderJSON(result)
			}
		}

	case "invoice.paid":
		//
		//
		//
		// // Extract data from the response
		var customerInvoice stripe.Invoice
		var prevCustomerInvoice stripe.Invoice

		//Get Previous Attribute
		jsonbody, err := json.Marshal(event.Data.PreviousAttributes)
		if err != nil {
			// revel.AppLog.Errorf("Cannot parse webhook JSON PreviousAttributes. EVENT: %v", event.Type)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "241"
			return c.RenderJSON(result)
		}

		// unmarshal new subscription values
		err = json.Unmarshal(event.Data.Raw, &customerInvoice)
		if err != nil {
			// revel.AppLog.Errorf("Cannot parse webhook JSON customerSubscription. EVENT: %v", event.Type)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "252"
			return c.RenderJSON(result)
		}

		// unmarshal previous subscription values
		err = json.Unmarshal(jsonbody, &prevCustomerInvoice)
		if err != nil {
			// revel.AppLog.Errorf("Cannot unmarshal previous attributes. EVENT: %v", event.Type)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "263"
			return c.RenderJSON(result)
		}

		//////////////////////////////
		// FETCH DATA
		//////////////////////////////

		//getSubscriptionBySubscriptionID
		// localSub, err := ops.GetSubscriptionBySubscriptionID(customerInvoice.Subscription.ID)
		localSub, err := ops.GetSubscriptionByStripeSubscriptionID(customerInvoice.Subscription.ID)
		if err != nil {
			// revel.AppLog.Errorf("Cannot get local subscription information. EVENT: %v SUB-ID: %v", event.Type, localSub.SubscriptionID)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "278"
			return c.RenderJSON(result)
		}

		// get user by id
		u, opsError := ops.GetUserByID(localSub.UserID)
		if opsError != nil {
			// revel.AppLog.Errorf("Cannot get user information. EVENT: %v SUB-ID: %v", event.Type, localSub.SubscriptionID)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "289"
			return c.RenderJSON(result)
		}

		//get company by id
		cus, opsError := ops.GetCompanyByID(localSub.CompanyID)
		if opsError != nil {
			// revel.AppLog.Errorf("Cannot get company information. EVENT: %v SUB-ID: %v", event.Type, localSub.SubscriptionID)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "300"
			return c.RenderJSON(result)
		}

		// get customer information params
		getCustomerParams := &stripe.CustomerParams{}
		getCustomerParams.AddExpand("subscriptions.data")
		getCustomerParams.AddExpand("invoice_settings.default_payment_method")

		// get customer
		cust, err := customer.Get(customerInvoice.Customer.ID, getCustomerParams)
		if err != nil {
			// revel.AppLog.Errorf("Cannot get customer information. EVENT: %v SUB-ID: %v", event.Type, localSub.SubscriptionID)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "316"
			return c.RenderJSON(result)
		}

		// get customer
		subscription, err := sub.Get(customerInvoice.Subscription.ID, nil)
		if err != nil {
			// revel.AppLog.Errorf("Cannot get customer information. EVENT: %v SUB-ID: %v", event.Type, localSub.SubscriptionID)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "327"
			return c.RenderJSON(result)
		}

		if cust.InvoiceSettings.DefaultPaymentMethod != nil {
			//////////////////////////////
			// Send Emails
			//////////////////////////////

			// Generate date with correct format
			startDate := time.Unix(customerInvoice.PeriodStart, 0)
			endDate := time.Unix(customerInvoice.PeriodEnd, 0)
			dateNow := time.Now()
			formatDateNow := dateNow.Format("January 2, 2006")
			formatStartDate := startDate.Format("January 2, 2006")
			formatEndDate := endDate.Format("January 2, 2006")

			var subType string

			// generate monthly or yearly
			switch string(subscription.Plan.Interval) {
			case "year":
				subType = constants.BILLING_TYPE_YEARLY
			case "month":
				subType = constants.BILLING_TYPE_MONTHLY
			default:
				break
			}

			var Recipients []mail.Recipient
			frontendUrl, _ := revel.Config.String("url.frontend")

			// Set set email recipients
			Recipient := mail.Recipient{
				ReceiptName:   u.FirstName,
				CompanyName:   cus.CompanyName,
				InvoiceNumber: customerInvoice.Number,
				Card:          cust.InvoiceSettings.DefaultPaymentMethod.Card.Brand,
				CardNumber:    cust.InvoiceSettings.DefaultPaymentMethod.Card.Last4,
				Email:         u.Email,
				DateNow:       formatDateNow,
				StartDate:     formatStartDate,
				EndDate:       formatEndDate,
				Plan:          strings.ToUpper(subType),
				Price:         strconv.FormatInt(customerInvoice.AmountPaid/100, 10), // divide by 100 https://youtu.be/1XKRxeo9414?t=316
				TotalPrice:    strconv.FormatInt(customerInvoice.AmountPaid/100, 10), // divide by 100 https://youtu.be/1XKRxeo9414?t=316
				Quantity:      strconv.FormatInt(customerInvoice.Lines.Data[0].Quantity, 10),
				RedirectLink:  frontendUrl,
			}

			// check if current seats is greater than previous seats, if so, send an email
			if subscription.Quantity > 5 && customerInvoice.AmountPaid != 0 {
				Recipients = append(Recipients, Recipient)

				jobs.Now(mail.SendEmail{
					Subject:    "We have received your payment",
					Recipients: Recipients,
					Template:   "receipt.html",
				})
			}

		}

	case "payment_intent.payment_failed":
		//
		//
		//
		var paymentIntent stripe.PaymentIntent

		err := json.Unmarshal(event.Data.Raw, &paymentIntent)
		if err != nil {
			// revel.AppLog.Errorf("Error parsing webhook JSON. EVENT: %v", event.Type)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "402"
			return c.RenderJSON(result)
		}

		piParams := &stripe.PaymentIntentParams{}
		piParams.AddExpand("customer")

		paymentInformation, err := paymentintent.Get(paymentIntent.ID, piParams)
		if err != nil {
			// revel.AppLog.Errorf("Cannot get payment intent information. EVENT: %v", event.Type)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "415"
			return c.RenderJSON(result)
		}
		//getCustomer
		customer, err := ops.GetUserByCustomerID(paymentIntent.Customer.ID)
		// subscription, err := ops.GetSubscriptionByCustomerID(paymentInformation.Customer.ID)
		if err != nil {
			// revel.AppLog.Errorf("Cannot get local subscription info. EVENT: %v CUS-ID: %v", event.Type, customer.CustomerID)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "426"
			return c.RenderJSON(result)
		}
		// get user by id
		u, opsError := ops.GetUserByID(customer.UserID)
		if opsError != nil {
			// revel.AppLog.Errorf("Cannot get user information. EVENT: %v USER-ID: %v", event.Type, u.UserID)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "436"
			return c.RenderJSON(result)
		}
		//getSubscriptionByCustomerID
		subscription, err := ops.GetSubscriptionInfo(u.ActiveCompany, customer.UserID)
		if err != nil {
			// revel.AppLog.Errorf("Cannot get local subscription info. EVENT: %v SUB-ID: %v", event.Type, subscription.SubscriptionID)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "446"
			return c.RenderJSON(result)
		}

		//get company by id
		cus, opsError := ops.GetCompanyByID(u.ActiveCompany)
		if opsError != nil {
			// revel.AppLog.Errorf("Cannot get company information. EVENT: %v SUB-ID: %v", event.Type, subscription.SubscriptionID)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "457"
			return c.RenderJSON(result)
		}

		integerStartDate, _ := strconv.ParseInt(subscription.StartDate, 10, 64)
		integerEndDate, _ := strconv.ParseInt(subscription.EndDate, 10, 64)
		startDate := time.Unix(integerStartDate, 0)
		endDate := time.Unix(integerEndDate, 0)
		dateNow := time.Now()
		formatDateNow := dateNow.Format("January 2, 2006")
		formatStartDate := startDate.Format("January 2, 2006")
		formatEndDate := endDate.Format("January 2, 2006")

		var Recipients []mail.Recipient
		frontendUrl, _ := revel.Config.String("url.frontend")
		Recipient := mail.Recipient{
			ReceiptName:  u.FirstName,
			CompanyName:  cus.CompanyName,
			Card:         paymentInformation.Charges.Data[0].PaymentMethodDetails.Card.Brand,
			CardNumber:   paymentInformation.Charges.Data[0].PaymentMethodDetails.Card.Last4,
			Email:        u.Email,
			DateNow:      formatDateNow,
			StartDate:    formatStartDate,
			EndDate:      formatEndDate,
			Plan:         strings.ToUpper(subscription.SubscriptionType),
			Price:        strconv.FormatFloat(float64(paymentInformation.Amount)/100, 'f', 2, 64),
			Quantity:     subscription.NoOfUsers,
			RedirectLink: frontendUrl,
		}

		Recipients = append(Recipients, Recipient)

		jobs.Now(mail.SendEmail{
			Subject:    "Your payment was declined",
			Recipients: Recipients,
			Template:   "payment-failed.html",
		})

		revel.AppLog.Infof("Success. EVENT: %v SUB-ID: %v", event.Type, subscription.SubscriptionID)

	case "customer.subscription.updated":
		//
		//
		//
		// // Extract data from the response
		var customerSubscription stripe.Subscription
		var prevSubscription stripe.Subscription

		//Get Previous Attribute
		jsonbody, err := json.Marshal(event.Data.PreviousAttributes)
		if err != nil {
			// revel.AppLog.Errorf("Cannot parse webhook JSON PreviousAttributes. EVENT: %v", event.Type)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "512"
			return c.RenderJSON(result)
		}

		// unmarshal new subscription values
		err = json.Unmarshal(event.Data.Raw, &customerSubscription)
		if err != nil {
			// revel.AppLog.Errorf("Cannot parse webhook JSON customerSubscription. EVENT: %v", event.Type)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "523"
			return c.RenderJSON(result)
		}

		// unmarshal previous subscription values
		err = json.Unmarshal(jsonbody, &prevSubscription)
		if err != nil {
			// revel.AppLog.Errorf("Cannot unmarshal previous attributes. EVENT: %v", event.Type)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "534"
			return c.RenderJSON(result)
		}

		//subscription data shorthand
		subscriptionData := customerSubscription.Items.Data[0]
		//

		//getSubscriptionBySubscriptionID
		// sub, err := ops.GetSubscriptionBySubscriptionID(subscriptionData.Subscription)
		sub, err := ops.GetSubscriptionByStripeSubscriptionID(subscriptionData.Subscription)
		//
		if err != nil {
			// revel.AppLog.Errorf("Cannot get local subscription information. EVENT: %v SUB-ID: %v", event.Type, sub.SubscriptionID)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "552"
			return c.RenderJSON(result)
		}

		//

		// check if it's previously a trial version
		var fromTrial string
		if prevSubscription.Status == "trialing" {
			fromTrial = constants.BOOL_TRUE
		} else {
			fromTrial = constants.BOOL_FALSE
		}

		// set update parameters
		params := models.Subscription{
			SubscriptionID:    sub.SubscriptionID,
			CompanyID:         sub.CompanyID,
			NoOfUsers:         strconv.FormatInt(subscriptionData.Quantity, 10),
			BillingType:       string(subscriptionData.Price.ID),
			Price:             strconv.FormatInt(subscriptionData.Price.UnitAmount, 10),
			Status:            strings.ToUpper(string(customerSubscription.Status)),
			EndDate:           strconv.FormatInt(customerSubscription.CurrentPeriodEnd, 10),
			CancelAtPeriodEnd: strings.ToUpper(strconv.FormatBool(customerSubscription.CancelAtPeriodEnd)),
			FromTrial:         fromTrial,
			SentTrialWarning:  "FALSE", //Remove this if we will use customer.subscription.trial_will_end event
			IsUpdated:         constants.BOOL_TRUE,
		}

		//
		updateSubscriptionError := ops.UpdateLocalSubscription(params)
		//
		if updateSubscriptionError != nil {
			// revel.AppLog.Errorf("Fail to update subscription. EVENT: %v SUB-ID: %v", event.Type, sub.SubscriptionID, updateSubscriptionError)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "588"
			return c.RenderJSON(result)
		}

		//////////////////////////////
		// FETCH DATA
		//////////////////////////////

		// get user by id
		u, opsError := ops.GetUserByID(sub.UserID)
		if opsError != nil {
			// revel.AppLog.Errorf("Cannot get user information. EVENT: %v SUB-ID: %v", event.Type, sub.SubscriptionID)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "603"
			return c.RenderJSON(result)
		}

		//get company by id
		cus, opsError := ops.GetCompanyByID(sub.CompanyID)
		if opsError != nil {
			// revel.AppLog.Errorf("Cannot get company information. EVENT: %v SUB-ID: %v", event.Type, sub.SubscriptionID)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "614"
			return c.RenderJSON(result)
		}
		_ = cus

		// get customer information params
		getCustomerParams := &stripe.CustomerParams{}
		getCustomerParams.AddExpand("subscriptions.data")
		getCustomerParams.AddExpand("invoice_settings.default_payment_method")

		// get customer
		cust, err := customer.Get(customerSubscription.Customer.ID, getCustomerParams)
		if err != nil {
			// revel.AppLog.Errorf("Cannot get customer information. EVENT: %v SUB-ID: %v", event.Type, sub.SubscriptionID)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "630"
			return c.RenderJSON(result)
		}
		_ = cust

		// get invoice
		inv, err := invoice.Get(customerSubscription.LatestInvoice.ID, nil)
		if err != nil {
			// revel.AppLog.Errorf("Cannot get invoice information. EVENT: %v SUB-ID: %v", event.Type, sub.SubscriptionID)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "641"
			return c.RenderJSON(result)
		}
		_ = inv
		//
		//

		// if cust.InvoiceSettings.DefaultPaymentMethod != nil {
		// 	//////////////////////////////
		// 	// Send Emails
		// 	//////////////////////////////

		// 	// Generate date with correct format
		// 	startDate := time.Unix(customerSubscription.CurrentPeriodStart, 0)
		// 	endDate := time.Unix(customerSubscription.CurrentPeriodEnd, 0)
		// 	dateNow := time.Now()
		// 	formatDateNow := dateNow.Format("Jan 2, 2006")
		// 	formatStartDate := startDate.Format("Jan 2, 2006")
		// 	formatEndDate := endDate.Format("Jan 2, 2006")

		// 	var subType string
		// 	var pricing string

		// 	p := message.NewPrinter(language.English)
		// 	newQuanity := customerSubscription.Quantity - prevSubscription.Quantity

		// 	// TODO: update
		// 	// generate monthly or yearly
		// 	switch string(subscriptionData.Plan.Interval) {
		// 	case "year":
		// 		subType = constants.BILLING_TYPE_YEARLY
		// 		pricing = p.Sprintf("%d\n", (newQuanity * int64(constants.PRICE[subscriptionData.Plan.ID]) * 12))
		// 	case "month":
		// 		subType = constants.BILLING_TYPE_MONTHLY
		// 		pricing = p.Sprintf("%d\n", (newQuanity * int64(constants.PRICE[subscriptionData.Plan.ID])))

		// 	default:
		// 		break
		// 	}

		// 	_ = pricing
		// 	var Recipients []mail.Recipient

		// 	// Set set email recipients
		// 	Recipient := mail.Recipient{
		// 		ReceiptName:   u.FirstName,
		// 		CompanyName:   cus.CompanyName,
		// 		InvoiceNumber: inv.Number,
		// 		Card:          cust.InvoiceSettings.DefaultPaymentMethod.Card.Brand,
		// 		CardNumber:    cust.InvoiceSettings.DefaultPaymentMethod.Card.Last4,
		// 		Email:         u.Email,
		// 		DateNow:       formatDateNow,
		// 		StartDate:     formatStartDate,
		// 		EndDate:       formatEndDate,
		// 		Plan:          strings.ToUpper(subType),
		// 		Price:         strconv.FormatInt(inv.AmountPaid/100, 10),
		// 		TotalPrice:    strconv.FormatInt(inv.AmountPaid/100, 10),
		// 		Quantity:      strconv.FormatInt(newQuanity, 10),
		// 	}

		// 	// send an email if they cancelled their subscription.
		// 	if customerSubscription.CancelAtPeriodEnd {
		// 		Recipients = append(Recipients, Recipient)

		// 		jobs.Now(mail.SendEmail{
		// 			Subject:    "We're sorry to see you go",
		// 			Recipients: Recipients,
		// 			Template:   "unsubscribe.html",
		// 		})
		// 	}

		// 	if customerSubscription.Status != "trialing" && customerSubscription.Quantity > prevSubscription.Quantity {
		// 		Recipients = append(Recipients, Recipient)

		// 		jobs.Now(mail.SendEmail{
		// 			Subject:    "Thank you for purchasing additional seats",
		// 			Recipients: Recipients,
		// 			Template:   "additional_seats.html",
		// 		})
		// 	}
		// }

		// TODO: check if subscription is cancelled??
		// 	if customerSubscription.CancelAtPeriodEnd {
		// 		Recipients = append(Recipients, Recipient)

		// 		jobs.Now(mail.SendEmail{
		// 			Subject:    "We're sorry to see you go",
		// 			Recipients: Recipients,
		// 			Template:   "unsubscribe.html",
		// 		})
		// 	}

		// check if there are changes on the seats
		if customerSubscription.Quantity != prevSubscription.Quantity {
			// eDate := time.Unix(customerSubscription.Created, 0)
			if customerSubscription.Quantity > 5 {
				eDate := time.Unix(inv.Created, 0)
				formatEDate := eDate.Format("January 2, 2006")
				recepient := mail.Recipient{
					Name:          u.FirstName,
					Plan:          subscriptionData.Plan.Nickname,
					PrevNumSeats:  prevSubscription.Quantity,
					NewNumSeats:   customerSubscription.Quantity,
					CompanyName:   cus.CompanyName,
					EffectiveDate: formatEDate,
					Email:         u.Email,
					IsFreePlan:    customerSubscription.Quantity <= 5,
				}

				subject := "Thank you for purchasing additional seats"
				template := "seats_updated.html"

				if customerSubscription.Quantity < prevSubscription.Quantity {
					subject = "Your subscription has been updated"

				}

				if customerSubscription.Quantity <= 5 && subscriptionData.Plan.Nickname == "Monthly Plan" {
					subject = "Your monthly subscription has been renewed successfully"
					template = "free_monthly_subscription_renewed.html"
				}

				jobs.Now(mail.SendEmail{
					Subject:    subject,
					Recipients: []mail.Recipient{recepient},
					Template:   template,
				})
			}
		}

	case "customer.subscription.deleted":
		//
		//
		//
		// // Extract data from the response
		var customerSubscription stripe.Subscription
		var prevSubscription stripe.Subscription

		//Get Previous Attribute
		jsonbody, err := json.Marshal(event.Data.PreviousAttributes)
		if err != nil {
			// revel.AppLog.Errorf("Cannot parse webhook JSON PreviousAttributes. EVENT: %v", event.Type)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "732"
			return c.RenderJSON(result)
		}

		// unmarshal new subscription values
		err = json.Unmarshal(event.Data.Raw, &customerSubscription)
		if err != nil {
			// revel.AppLog.Errorf("Cannot parse webhook JSON customerSubscription. EVENT: %v", event.Type)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "743"
			return c.RenderJSON(result)
		}

		// unmarshal previous subscription values
		err = json.Unmarshal(jsonbody, &prevSubscription)
		if err != nil {
			// revel.AppLog.Errorf("Cannot unmarshal previous attributes. EVENT: %v", event.Type)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "754"
			return c.RenderJSON(result)
		}

		//subscription data shorthand
		subscriptionData := customerSubscription.Items.Data[0]

		//getSubscriptionBySubscriptionID
		sub, err := ops.GetSubscriptionByStripeSubscriptionID(subscriptionData.Subscription)
		if err != nil {
			// revel.AppLog.Errorf("Cannot get local subscription information. EVENT: %v SUB-ID: %v", event.Type, sub.SubscriptionID)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "768"
			return c.RenderJSON(result)
		}
		//
		// set update parameters
		params := models.Subscription{
			CompanyID:      sub.CompanyID,
			SubscriptionID: sub.SubscriptionID,
			NoOfUsers:      strconv.FormatInt(subscriptionData.Quantity, 10),
			// BillingType:       strings.ToUpper(string(subscriptionData.Plan.Interval)),
			BillingType:       string(subscriptionData.Price.ID),
			Price:             strconv.FormatInt(subscriptionData.Price.UnitAmount, 10),
			Status:            strings.ToUpper(string(customerSubscription.Status)),
			EndDate:           strconv.FormatInt(customerSubscription.CurrentPeriodEnd, 10),
			CancelAtPeriodEnd: strings.ToUpper(strconv.FormatBool(customerSubscription.CancelAtPeriodEnd)),
			FromTrial:         constants.BOOL_FALSE,
			SentTrialWarning:  "FALSE", //Remove this if we will use customer.subscription.trial_will_end event
			IsUpdated:         constants.BOOL_TRUE,
		}

		updateSubscriptionError := ops.UpdateLocalSubscription(params)
		if updateSubscriptionError != nil {

			//
			// revel.AppLog.Errorf("Fail to update subscription. EVENT: %v SUB-ID: %v", event.Type, sub.SubscriptionID)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "796"
			return c.RenderJSON(result)
		}

	case "customer.subscription.trial_will_end":
		//
		//
		//
		// // Extract data from the response
		var customerSubscription stripe.Subscription
		var prevSubscription stripe.Subscription

		//Get Previous Attribute
		jsonbody, err := json.Marshal(event.Data.PreviousAttributes)
		if err != nil {
			// revel.AppLog.Errorf("Cannot parse webhook JSON PreviousAttributes. EVENT: %v", event.Type)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "815"
			return c.RenderJSON(result)
		}

		// unmarshal new subscription values
		err = json.Unmarshal(event.Data.Raw, &customerSubscription)
		if err != nil {
			// revel.AppLog.Errorf("Cannot parse webhook JSON customerSubscription. EVENT: %v", event.Type)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "826"
			return c.RenderJSON(result)
		}

		// unmarshal previous subscription values
		err = json.Unmarshal(jsonbody, &prevSubscription)
		if err != nil {
			// revel.AppLog.Errorf("Cannot unmarshal previous attributes. EVENT: %v", event.Type)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "837"
			return c.RenderJSON(result)
		}

		if customerSubscription.Items.Data[0] == nil {
			// revel.AppLog.Errorf("Customet Subscrition Item not found. EVENT: %v", event.Type)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "846"
			return c.RenderJSON(result)
		}

		//subscription data shorthand
		subscriptionData := customerSubscription.Items.Data[0]

		//getSubscriptionBySubscriptionID
		// sub, err := ops.GetSubscriptionBySubscriptionID(subscriptionData.Subscription)
		sub1, err := ops.GetSubscriptionByStripeSubscriptionID(subscriptionData.Subscription)
		if err != nil {
			// revel.AppLog.Errorf("Cannot get local subscription information. EVENT: %v SUB-ID: %v", event.Type, sub1.SubscriptionID)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "861"
			return c.RenderJSON(result)
		}

		//////////////////////////////
		// FETCH DATA
		//////////////////////////////

		subscription, err := sub.Get(subscriptionData.Subscription, nil)
		if err != nil {
			// revel.AppLog.Errorf("Cannot get customer information. EVENT: %v SUB-ID: %v", event.Type, subscriptionData.Subscription)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "327"
			return c.RenderJSON(result)
		}

		if subscription.Quantity > 5 {

			u, opsError := ops.GetUserByID(sub1.UserID)
			if opsError != nil {
				// revel.AppLog.Errorf("Cannot get user information. EVENT: %v SUB-ID: %v", event.Type, sub1.SubscriptionID)
				c.Response.Status = http.StatusBadRequest
				result["error"] = err
				result["event"] = event.Type
				result["line"] = "876"
				return c.RenderJSON(result)
			}

			var Recipients []mail.Recipient

			// Set set email recipients
			Recipient := mail.Recipient{
				Name:         u.FirstName + " " + u.LastName,
				Email:        u.Email,
				RedirectLink: frontendUrl + "/my-profile?slug=payment-billing",
			}

			Recipients = append(Recipients, Recipient)

			//

			jobs.Now(mail.SendEmail{
				Subject:    "Your SaaSConsole Trial Is Expiring Soon",
				Recipients: Recipients,
				Template:   "seven_days_trial_expiry.html",
			})
		}

	case "payment_method.attached":
		//
		//
		//
		// // Extract data from the response
		var paymentMethod stripe.PaymentMethod
		var prevPaymentMethod stripe.PaymentMethod

		//Get Previous Attribute
		jsonbody, err := json.Marshal(event.Data.PreviousAttributes)
		if err != nil {
			// revel.AppLog.Errorf("Cannot parse webhook JSON PreviousAttributes. EVENT: %v", event.Type)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "912"
			return c.RenderJSON(result)
		}

		// unmarshal new subscription values
		err = json.Unmarshal(event.Data.Raw, &paymentMethod)
		if err != nil {
			// revel.AppLog.Errorf("Cannot parse webhook JSON paymentMethod. EVENT: %v", event.Type)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "923"
			return c.RenderJSON(result)
		}

		// unmarshal previous subscription values
		err = json.Unmarshal(jsonbody, &prevPaymentMethod)
		if err != nil {
			// revel.AppLog.Errorf("Cannot unmarshal previous attributes. EVENT: %v", event.Type)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "934"
			return c.RenderJSON(result)
		}

		// get customer information params
		getPaymentMethodParams := &stripe.PaymentMethodParams{}
		getPaymentMethodParams.AddExpand("customer")

		// get customer
		pm, err := paymentmethod.Get(paymentMethod.ID, getPaymentMethodParams)
		if err != nil {
			// revel.AppLog.Errorf("Cannot get payment method information. EVENT: %v PM-ID: %v", event.Type, paymentMethod.ID)
			c.Response.Status = http.StatusBadRequest
			result["error"] = err
			result["event"] = event.Type
			result["line"] = "949"
			return c.RenderJSON(result)
		}

		customerID := pm.Customer.ID

		subListParams := &stripe.SubscriptionListParams{}
		subListParams.Filters.AddFilter("limit", "", "1")
		subListParams.Filters.AddFilter("status", "", "past_due")
		subListParams.Filters.AddFilter("customer", "", customerID)

		subs := sub.List(subListParams)
		for subs.Next() {
			sub := subs.Subscription()

			if sub.ID != "" {
				invoiceListParams := &stripe.InvoiceListParams{}
				invoiceListParams.Filters.AddFilter("limit", "", "1")
				invoiceListParams.Filters.AddFilter("status", "", "open")
				invoiceListParams.Filters.AddFilter("customer", "", customerID)
				invoiceListParams.Filters.AddFilter("subscription", "", sub.ID)

				invoiceList := invoice.List(invoiceListParams)
				for invoiceList.Next() {
					iv := invoiceList.Invoice()
					if iv.ID != "" {
						in, _ := invoice.Pay(
							iv.ID,
							nil,
						)
						_ = in
						//
					}
				}
			}
		}

	default:
		// revel.AppLog.Errorf("Unhandled event. EVENT: %v", event.Type)
		c.Response.Status = http.StatusOK
		result["event"] = event.Type
		result["line"] = "991"
		return c.RenderJSON(result)
	}

	c.Response.Status = http.StatusOK
	return nil
}
