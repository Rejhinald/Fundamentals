package controllers

import (
	"encoding/json"
	"grooper/app/constants"
	stripeoperations "grooper/app/integrations/stripe"
	"grooper/app/models"
	stripeModels "grooper/app/models/stripe"
	ops "grooper/app/operations"
	"grooper/app/utils"
	"strconv"

	"github.com/revel/revel"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/sub"
)

type StripeControllerz struct {
	*revel.Controller
}

func (c StripeControllerz) Test() revel.Result {
	// userID := c.ViewArgs["userID"].(string)

	subid := c.Params.Query.Get("sub_id")

	var update bool
	var trialEnd int64
	c.Params.Bind(&trialEnd, "trial_end")
	c.Params.Bind(&update, "update")

	if !update {
		subscription, _ := sub.Get(subid, &stripe.SubscriptionParams{})
		c.Response.Status = 200
		return c.RenderJSON(subscription)
	}

	params := &stripe.SubscriptionParams{
		TrialEnd: stripe.Int64(trialEnd),
	}

	subscription, err := sub.Update(subid, params)
	if err != nil {
		return c.RenderJSON(err)

	}
	return c.RenderJSON(subscription)
}

func (c StripeControllerz) UpdateSubscriptionCustom() revel.Result {
	companyID := c.ViewArgs["companyID"].(string)
	userID := c.ViewArgs["userID"].(string)

	var option string
	c.Params.Bind(&option, "option")
	//
	switch option {
	case "UPDATE_TRIAL":
		var trialEnd int64
		c.Params.Bind(&trialEnd, "trial_end")

		if trialEnd == 0 {
			c.Response.Status = 422
			return c.RenderJSON([]revel.ValidationError{
				{
					Key:     "trial_end",
					Message: "Tiral end ID is required.",
				},
			})
		}

		gucs := ops.GetUserCompanySubscriptionParams{UserID: userID, CompanyID: companyID}
		saasSubscription, customeError := ops.GetUserCompanySubscription(gucs, c.Controller)
		if customeError != nil {
			c.Response.Status = customeError.HTTPStatusCode
			return c.RenderJSON(customeError)
		}

		if saasSubscription.StripeSubscriptionID == nil || *saasSubscription.StripeSubscriptionID == "" {
			return c.RenderJSON(models.ErrorResponse{
				HTTPStatusCode: 422,
				Message:        "Stripe Subscription ID is missing.",
				Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_422),
			})
		}

		params := &stripe.SubscriptionParams{
			TrialEnd: stripe.Int64(trialEnd),
			// BillingCycleAnchor: &trialEnd,
			// BillingThresholds: &stripe.SubscriptionBillingThresholdsParams{
			// 	ResetBillingCycleAnchor: aws.Bool(true),
			// },
		}

		subscriptionStripe, err := sub.Update(*saasSubscription.StripeSubscriptionID, params)
		if err != nil {
			c.Response.Status = 500
			return c.RenderJSON(err)

		}
		c.Response.Status = 200
		return c.RenderJSON(subscriptionStripe)

	case "SUSPEND_SUBSCRIPTION":

		gucs := ops.GetUserCompanySubscriptionParams{UserID: userID, CompanyID: companyID}
		saasSubscription, customeError := ops.GetUserCompanySubscription(gucs, c.Controller)
		if customeError != nil {
			c.Response.Status = customeError.HTTPStatusCode
			return c.RenderJSON(customeError)
		}

		if saasSubscription.StripeSubscriptionID == nil || *saasSubscription.StripeSubscriptionID == "" {
			return c.RenderJSON(models.ErrorResponse{
				HTTPStatusCode: 422,
				Message:        "Stripe Subscription ID is missing.",
				Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_422),
			})
		}

		s, err := sub.Cancel(*saasSubscription.StripeSubscriptionID, nil)
		if err != nil {
			c.Response.Status = 500
			return c.RenderJSON(err)
		}

		c.Response.Status = 200
		return c.RenderJSON(s)
	case "GET_SUBSCRIPTION":
		gucs := ops.GetUserCompanySubscriptionParams{UserID: userID, CompanyID: companyID}
		saasSubscription, customeError := ops.GetUserCompanySubscription(gucs, c.Controller)
		if customeError != nil {
			c.Response.Status = customeError.HTTPStatusCode
			return c.RenderJSON(customeError)
		}

		if saasSubscription.StripeSubscriptionID == nil || *saasSubscription.StripeSubscriptionID == "" {
			return c.RenderJSON(models.ErrorResponse{
				HTTPStatusCode: 422,
				Message:        "Stripe Subscription ID is missing.",
				Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_422),
			})
		}
		params := &stripe.SubscriptionParams{}
		params.AddExpand("customer")
		s, err := sub.Get(*saasSubscription.StripeSubscriptionID, params)
		if err != nil {
			c.Response.Status = 500
			return c.RenderJSON(err)
		}

		c.Response.Status = 200
		return c.RenderJSON(s)

	default:
		//
	}

	c.Response.Status = 204
	return nil
}

func (c StripeControllerz) GetStripePricing() revel.Result {
	prices, stripeErr := stripeoperations.GetStripePrices()
	if stripeErr != nil {
		c.Response.Status = stripeErr.HTTPStatusCode
		return c.RenderJSON(stripeErr)
	}
	c.Response.Status = 200
	return c.RenderJSON(prices)
}

func (c StripeControllerz) CreateStripeSubscription() revel.Result {
	userID := c.ViewArgs["userID"].(string)

	cardId := c.Params.Form.Get("card_id")
	priceID := c.Params.Form.Get("price_id")
	quantityGet := c.Params.Form.Get("quantity")

	if utils.FindEmptyStringElement([]string{priceID, quantityGet}) {
		c.Response.Status = 422
		return c.RenderJSON(map[string]interface{}{
			"message": "Missing parameters",
			"status":  utils.GetHTTPStatus(constants.HTTP_STATUS_422),
		})
	}

	quantity, err := strconv.ParseInt(quantityGet, 10, 64)
	if err != nil {
		c.Response.Status = 422
		return c.RenderJSON(map[string]interface{}{
			"message": "Invalid quantity value",
			"status":  utils.GetHTTPStatus(constants.HTTP_STATUS_422),
		})
	}

	companies, err := ops.GetUserSubscribedCompanies(userID)
	if err != nil {
		c.Response.Status = 500
		return c.RenderJSON(map[string]interface{}{
			"message": "Can't get user subscribed companies",
			"status":  utils.GetHTTPStatus(constants.HTTP_STATUS_500),
		})
	}

	userInfo, stripeError := ops.GetUserCustomerID(userID)
	if stripeError != nil {
		c.Response.Status = stripeError.HTTPStatusCode
		return c.RenderJSON(stripeError)
	}

	companyCount := len(companies)

	//Add payment method if the company subscripton of user is greater than or equal 1
	if companyCount > 0 {
		if utils.FindEmptyStringElement([]string{cardId}) {
			c.Response.Status = 422
			return c.RenderJSON(map[string]interface{}{
				"message": "Missing parameters",
				"status":  utils.GetHTTPStatus(constants.HTTP_STATUS_422),
			})
		}

		cardExisting, stripeError := stripeoperations.CheckCustomerPaymentMethod(*userInfo.CustomerID, cardId)
		if stripeError != nil {
			c.Response.Status = stripeError.HTTPStatusCode
			return c.RenderJSON(stripeError)
		}

		if *cardExisting {

			// c.Response.Status = 400
			// return c.RenderJSON("cardExisting: " + "customerId:" + *userInfo.CustomerID + ";cardId:" + cardId)
			cus, stripeError := stripeoperations.UpdateStripeCustomerDefaultCard(*userInfo.CustomerID, cardId)
			if stripeError != nil {
				c.Response.Status = stripeError.HTTPStatusCode
				return c.RenderJSON(stripeError)
			}
			_ = cus
		} else {
			pm, stripeError := stripeoperations.AttachStripeCardToCustomer(*userInfo.CustomerID, cardId)
			if stripeError != nil {
				c.Response.Status = stripeError.HTTPStatusCode
				return c.RenderJSON(stripeError)
			}
			_ = pm

			cus, stripeError := stripeoperations.UpdateStripeCustomerDefaultCard(*userInfo.CustomerID, cardId)
			if stripeError != nil {
				c.Response.Status = stripeError.HTTPStatusCode
				return c.RenderJSON(stripeError)
			}
			_ = cus
		}

	}

	cratedSubscription, stripeError := stripeoperations.CreateStripeSubscriptionHelper(*userInfo.CustomerID, priceID, quantity, companyCount)
	if stripeError != nil {
		c.Response.Status = stripeError.HTTPStatusCode
		return c.RenderJSON(stripeError)
	}

	c.Response.Status = 200
	return c.RenderJSON(cratedSubscription)
}

func (c StripeControllerz) UpdateStripeSubscription() revel.Result {
	companyID := c.ViewArgs["companyID"].(string)
	userID := c.ViewArgs["userID"].(string)

	var cardID string
	c.Params.Bind(&cardID, "card_id")

	priceId := c.Params.Form.Get("price_id")
	quantityGet := c.Params.Form.Get("quantity")

	if utils.FindEmptyStringElement([]string{priceId, quantityGet}) {
		c.Response.Status = 422
		return c.RenderJSON(map[string]interface{}{
			"message": "Missing parameters",
			"status":  utils.GetHTTPStatus(constants.HTTP_STATUS_422),
		})
	}
	gucs := ops.GetUserCompanySubscriptionParams{UserID: userID, CompanyID: companyID}
	saasSubscription, customeError := ops.GetUserCompanySubscription(gucs, c.Controller)
	if customeError != nil {
		c.Response.Status = customeError.HTTPStatusCode
		return c.RenderJSON(customeError)
	}

	//IF CARD ID IS NOT EMPTY ATTACH CARD
	if cardID != "" {
		userInfo, stripeError := ops.GetUserCustomerID(saasSubscription.UserID)
		if stripeError != nil {
			c.Response.Status = stripeError.HTTPStatusCode
			return c.RenderJSON(stripeError)
		}

		cardExisting, stripeError := stripeoperations.CheckCustomerPaymentMethod(*userInfo.CustomerID, cardID)
		if stripeError != nil {
			c.Response.Status = stripeError.HTTPStatusCode
			return c.RenderJSON(stripeError)
		}

		if *cardExisting {
			cus, stripeError := stripeoperations.UpdateStripeCustomerDefaultCard(*userInfo.CustomerID, cardID)
			if stripeError != nil {
				c.Response.Status = stripeError.HTTPStatusCode
				return c.RenderJSON(stripeError)
			}
			_ = cus
		} else {
			pm, stripeError := stripeoperations.AttachStripeCardToCustomer(*userInfo.CustomerID, cardID)
			if stripeError != nil {
				c.Response.Status = stripeError.HTTPStatusCode
				return c.RenderJSON(stripeError)
			}
			_ = pm

			cus, stripeError := stripeoperations.UpdateStripeCustomerDefaultCard(*userInfo.CustomerID, cardID)
			if stripeError != nil {
				c.Response.Status = stripeError.HTTPStatusCode
				return c.RenderJSON(stripeError)
			}
			_ = cus
		}
	}

	if saasSubscription.StripeSubscriptionID == nil || *saasSubscription.StripeSubscriptionID == "" {
		c.Response.Status = 422
		return c.RenderJSON(map[string]interface{}{
			"message": "Stripe subscription id is missing",
			"status":  utils.GetHTTPStatus(constants.HTTP_STATUS_422),
		})
	}

	stripeSubscriptionId := *saasSubscription.StripeSubscriptionID

	quantity, defErr := strconv.ParseInt(quantityGet, 10, 64)
	if defErr != nil {
		c.Response.Status = 422
		return c.RenderJSON(map[string]interface{}{
			"message": "Invalid quantity value",
			"status":  utils.GetHTTPStatus(constants.HTTP_STATUS_422),
		})
	}

	stripeSubscription, stripeError := stripeoperations.GetStripeSubscription(stripeSubscriptionId)
	if (stripeError != nil && stripeError.HTTPStatusCode == 404) || stripeSubscription.Status == "canceled" {
		// get stripe user
		userInfo, stripeError := ops.GetUserCustomerID(saasSubscription.UserID)
		if stripeError != nil {
			c.Response.Status = stripeError.HTTPStatusCode
			return c.RenderJSON(stripeError)
		}
		// get user companies
		companies, err := ops.GetUserSubscribedCompanies(userID)
		if err != nil {
			c.Response.Status = 500
			return c.RenderJSON(map[string]interface{}{
				"message": "Can't get user subscribed companies",
				"status":  utils.GetHTTPStatus(constants.HTTP_STATUS_500),
			})
		}
		// create  subscription
		cratedSubscription, stripeError := stripeoperations.CreateStripeSubscriptionHelper(*userInfo.CustomerID, priceId, quantity, len(companies))
		if stripeError != nil {
			c.Response.Status = stripeError.HTTPStatusCode
			return c.RenderJSON(stripeError)
		}

		c.Response.Status = 200
		return c.RenderJSON(cratedSubscription)
		// attach to user subscription

	}

	if stripeError != nil {
		c.Response.Status = stripeError.HTTPStatusCode
		return c.RenderJSON(stripeError)

	}

	// utils.PrintJSON(stripeSubscription, "stripeSubscription")

	params := &stripe.SubscriptionParams{
		Items: []*stripe.SubscriptionItemsParams{
			{
				ID:       stripe.String(stripeSubscription.Items.Data[0].ID),
				Quantity: stripe.Int64(quantity),
				Price:    stripe.String(priceId),
			},
		},
		ProrationBehavior: stripe.String(string(stripe.SubscriptionProrationBehaviorAlwaysInvoice)),
		ProrationDate:     stripe.Int64(stripeSubscription.CurrentPeriodStart),
	}

	if quantity <= 5 {
		//remove free tiral if on trial
		params.TrialEndNow = stripe.Bool(true)
	}

	updatedSubscription, stripeError := stripeoperations.UpdateStripeSubscription(stripeSubscriptionId, params)
	if stripeError != nil {
		c.Response.Status = stripeError.HTTPStatusCode
		return c.RenderJSON(stripeError)
	}

	c.Response.Status = 200
	return c.RenderJSON(updatedSubscription)
}

func (c StripeControllerz) CancelStripeSubscription() revel.Result {
	companyID := c.ViewArgs["companyID"].(string)
	userID := c.ViewArgs["userID"].(string)

	gucs := ops.GetUserCompanySubscriptionParams{UserID: userID, CompanyID: companyID}
	saasSubscription, customeError := ops.GetUserCompanySubscription(gucs, c.Controller)
	if customeError != nil {
		c.Response.Status = customeError.HTTPStatusCode
		return c.RenderJSON(customeError)
	}

	if saasSubscription.StripeSubscriptionID == nil || *saasSubscription.StripeSubscriptionID == "" {
		c.Response.Status = 422
		return c.RenderJSON(map[string]interface{}{
			"message": "Stripe subscription id is missing",
			"status":  utils.GetHTTPStatus(constants.HTTP_STATUS_422),
		})
	}

	stripeSubscriptionId := *saasSubscription.StripeSubscriptionID

	cancelledSubscription, stripeError := stripeoperations.CancelStripeSubscription(stripeSubscriptionId)
	if stripeError != nil {
		c.Response.Status = stripeError.HTTPStatusCode
		return c.RenderJSON(stripeError)
	}

	c.Response.Status = 200
	return c.RenderJSON(cancelledSubscription)
}

func (c StripeControllerz) AddStripeCustomerCard() revel.Result {
	companyID := c.ViewArgs["companyID"].(string)
	userID := c.ViewArgs["userID"].(string)

	var cardID string
	var defaultCard bool

	c.Params.Bind(&cardID, "card_id")
	c.Params.Bind(&defaultCard, "default_card")

	if utils.FindEmptyStringElement([]string{cardID}) {
		c.Response.Status = 422
		return c.RenderJSON(map[string]interface{}{
			"message": "Missing parameters",
			"status":  utils.GetHTTPStatus(constants.HTTP_STATUS_422),
		})
	}

	gucs := ops.GetUserCompanySubscriptionParams{UserID: userID, CompanyID: companyID}
	saasSubscription, customeError := ops.GetUserCompanySubscription(gucs, c.Controller)
	if customeError != nil {
		c.Response.Status = customeError.HTTPStatusCode
		return c.RenderJSON(customeError)
	}

	userInfo, err := ops.GetUserCustomerID(saasSubscription.UserID)
	if err != nil {
		c.Response.Status = err.HTTPStatusCode
		return c.RenderJSON(err)
	}

	cardExisting, stripeError := stripeoperations.CheckCustomerPaymentMethod(*userInfo.CustomerID, cardID)
	if stripeError != nil {
		c.Response.Status = stripeError.HTTPStatusCode
		return c.RenderJSON(stripeError)
	}

	if *cardExisting {
		cus, stripeError := stripeoperations.UpdateStripeCustomerDefaultCard(*userInfo.CustomerID, cardID)
		if stripeError != nil {
			c.Response.Status = stripeError.HTTPStatusCode
			return c.RenderJSON(stripeError)
		}
		_ = cus
	} else {
		pm, stripeError := stripeoperations.AttachStripeCardToCustomer(*userInfo.CustomerID, cardID)
		if stripeError != nil {
			c.Response.Status = stripeError.HTTPStatusCode
			return c.RenderJSON(stripeError)
		}
		_ = pm

		cus, stripeError := stripeoperations.UpdateStripeCustomerDefaultCard(*userInfo.CustomerID, cardID)
		if stripeError != nil {
			c.Response.Status = stripeError.HTTPStatusCode
			return c.RenderJSON(stripeError)
		}
		_ = cus
	}

	customer, stripeError := stripeoperations.GetStripeCustomer(*userInfo.CustomerID)
	if stripeError != nil {
		c.Response.Status = stripeError.HTTPStatusCode
		return c.RenderJSON(stripeError)
	}

	c.Response.Status = 200
	return c.RenderJSON(customer)
}

func (c StripeControllerz) GetCustomerDetails() revel.Result {
	companyID := c.ViewArgs["companyID"].(string)
	userID := c.ViewArgs["userID"].(string)

	var priceList map[string]float64
	var prices, _ = revel.Config.String("stripe.price.list")
	jErr := json.Unmarshal([]byte(prices), &priceList)
	if jErr != nil {
		revel.AppLog.Fatal("Error unmarshaling PRICE JSON: ", jErr)
	}

	gucs := ops.GetUserCompanySubscriptionParams{UserID: userID, CompanyID: companyID}
	subscription, customeError := ops.GetUserCompanySubscription(gucs, c.Controller)
	if customeError != nil {
		c.Response.Status = customeError.HTTPStatusCode
		return c.RenderJSON(customeError)
	}

	if subscription.StripeSubscriptionID == nil || *subscription.StripeSubscriptionID == "" {
		c.Response.Status = 422
		return c.RenderJSON("*subscription.StripeSubscriptionID")
	}

	customerDetails := stripeModels.StripeCustomerDetails{}

	// return c.RenderJSON(subscription)

	stripeSubs, stripeError := stripeoperations.GetStripeSubscription(*subscription.StripeSubscriptionID)
	if stripeError != nil {
		c.Response.Status = stripeError.HTTPStatusCode
		return c.RenderJSON(stripeError)
	}

	customerDetails.Subscription = stripeModels.StripeSubscription{
		ID:                 stripeSubs.ID,
		Quantity:           int(stripeSubs.Quantity),
		Status:             string(stripeSubs.Status),
		CurrentPeriodEnd:   int(stripeSubs.CurrentPeriodEnd),
		CurrentPeriodStart: int(stripeSubs.CurrentPeriodStart),
		Plan: stripeModels.StripePrice{
			ID:       stripeSubs.Plan.ID,
			Nickname: stripeSubs.Plan.Nickname,
			// Amount:   constants.PRICE[stripeSubs.Plan.ID],
			Amount: priceList[stripeSubs.Plan.ID],
		},
	}

	userInfo, err := ops.GetUserByID(subscription.UserID)
	if err != nil {
		c.Response.Status = 500
		return c.RenderJSON(err)
	}

	customer, stripeError := stripeoperations.GetStripeCustomer(*userInfo.CustomerID)
	if stripeError != nil {
		c.Response.Status = stripeError.HTTPStatusCode
		return c.RenderJSON(stripeError)
	}

	customerDetails.Customer = stripeModels.StripeCustomer{
		ID:    customer.ID,
		Email: customer.Email,
		Name:  customer.Name,
	}

	var stripeCard *stripeModels.StripeCard

	if customer.InvoiceSettings.DefaultPaymentMethod != nil {
		stripeCard = &stripeModels.StripeCard{
			ID:       customer.InvoiceSettings.DefaultPaymentMethod.ID,
			Brand:    string(customer.InvoiceSettings.DefaultPaymentMethod.Card.Brand),
			ExpMonth: customer.InvoiceSettings.DefaultPaymentMethod.Card.ExpMonth,
			ExpYear:  customer.InvoiceSettings.DefaultPaymentMethod.Card.ExpYear,
			Country:  customer.InvoiceSettings.DefaultPaymentMethod.Card.Country,
			Funding:  string(customer.InvoiceSettings.DefaultPaymentMethod.Card.Funding),
			Last4:    customer.InvoiceSettings.DefaultPaymentMethod.Card.Last4,
		}
	}
	customerDetails.Card = stripeCard

	c.Response.Status = 200
	return c.RenderJSON(customerDetails)
}

func (c StripeControllerz) GetStripeCustomerCards() revel.Result {
	companyID := c.ViewArgs["companyID"].(string)
	userID := c.ViewArgs["userID"].(string)

	gucs := ops.GetUserCompanySubscriptionParams{UserID: userID, CompanyID: companyID}
	subscription, customeError := ops.GetUserCompanySubscription(gucs, c.Controller)
	if customeError != nil {
		c.Response.Status = customeError.HTTPStatusCode
		return c.RenderJSON(customeError)
	}

	userInfo, stripeError := ops.GetUserCustomerID(subscription.UserID)
	if stripeError != nil {
		c.Response.Status = stripeError.HTTPStatusCode
		return c.RenderJSON(stripeError)
	}

	cards, stripeError := stripeoperations.GetStripeCustomerCards(*userInfo.CustomerID)
	if stripeError != nil {
		c.Response.Status = stripeError.HTTPStatusCode
		return c.RenderJSON(stripeError)
	}

	c.Response.Status = 200
	return c.RenderJSON(cards)
}

func (c StripeControllerz) ReactivateStripeSubscription() revel.Result {
	companyID := c.ViewArgs["companyID"].(string)
	userID := c.ViewArgs["userID"].(string)

	gucs := ops.GetUserCompanySubscriptionParams{UserID: userID, CompanyID: companyID}
	saasSubscription, customeError := ops.GetUserCompanySubscription(gucs, c.Controller)
	if customeError != nil {
		c.Response.Status = customeError.HTTPStatusCode
		return c.RenderJSON(customeError)
	}

	priceId := saasSubscription.BillingType
	quantity, defErr := strconv.ParseInt(saasSubscription.NoOfUsers, 10, 64)
	if defErr != nil {
		c.Response.Status = 422
		return c.RenderJSON(map[string]interface{}{
			"message": "Invalid quantity value",
			"status":  utils.GetHTTPStatus(constants.HTTP_STATUS_422),
		})
	}
	stripeSubscriptionId := *saasSubscription.StripeSubscriptionID

	subscription, stripeError := stripeoperations.ReactivateCustomerSubscriptionHelper(priceId, stripeSubscriptionId, quantity)
	if stripeError != nil {
		c.Response.Status = stripeError.HTTPStatusCode
		return c.RenderJSON(stripeError)
	}

	c.Response.Status = 200
	return c.RenderJSON(subscription)
}

func (c StripeControllerz) GetCustomerInvoice() revel.Result {
	companyID := c.ViewArgs["companyID"].(string)
	userID := c.ViewArgs["userID"].(string)

	gucs := ops.GetUserCompanySubscriptionParams{UserID: userID, CompanyID: companyID}
	saasSubscription, customeError := ops.GetUserCompanySubscription(gucs, c.Controller)
	if customeError != nil {
		c.Response.Status = customeError.HTTPStatusCode
		return c.RenderJSON(customeError)
	}

	userInfo, stripeError := ops.GetUserCustomerID(saasSubscription.UserID)
	if stripeError != nil {
		c.Response.Status = stripeError.HTTPStatusCode
		return c.RenderJSON(stripeError)
	}

	if userInfo.CustomerID == nil || *userInfo.CustomerID == "" {
		c.Response.Status = 422
		return c.RenderJSON(models.ErrorResponse{
			Message: "User Customer id not found",
			Status:  utils.GetHTTPStatus(constants.HTTP_STATUS_422),
		})
	}

	invoice, stripeError := stripeoperations.GetCustomerPaidInvoice(*userInfo.CustomerID)
	if stripeError != nil {
		c.Response.Status = stripeError.HTTPStatusCode
		return c.RenderJSON(stripeError)
	}

	invData := []stripeModels.StripeCustomerPaidInvoice{}
	for _, i := range *invoice {
		invData = append(invData, stripeModels.StripeCustomerPaidInvoice{
			ReceiptNumber: i.Number,
			PDFUrl:        i.InvoicePDF,
			PaidAt:        int(i.StatusTransitions.PaidAt),
			PriceID:       i.Lines.Data[0].Price.ID,
			PeriodStart:   int(i.Lines.Data[0].Period.Start),
			PeriodEnd:     int(i.Lines.Data[0].Period.End),
			AmountPaid:    int(i.AmountPaid),
		})
	}

	c.Response.Status = 200
	return c.RenderJSON(invData)
}

// GetSubscription
func (c StripeControllerz) GetStripeSubscriptionStatus() revel.Result {
	companyID := c.ViewArgs["companyID"].(string)
	userID := c.ViewArgs["userID"].(string)

	data := make(map[string]interface{})

	gucs := ops.GetUserCompanySubscriptionParams{UserID: userID, CompanyID: companyID}
	saasSubscription, customeError := ops.GetUserCompanySubscription(gucs, c.Controller)
	if customeError != nil {
		c.Response.Status = customeError.HTTPStatusCode
		return c.RenderJSON(customeError)
	}

	// get customer information params
	getSubscriptionParams := &stripe.SubscriptionParams{}
	getSubscriptionParams.AddExpand("latest_invoice")
	getSubscriptionParams.AddExpand("latest_invoice.payment_intent")

	if saasSubscription.StripeSubscriptionID == nil {
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 422,
			Message:        "Stripe Subscription ID is missing.",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_422),
		})
	}

	// Get Stripe Subscription
	subscription, err := sub.Get(*saasSubscription.StripeSubscriptionID, getSubscriptionParams)
	if err != nil {
		c.Response.Status = 496
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 496,
			Message:        "Subscription not found.",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_496),
		})
	}

	// Check if payment_intent is nil
	if subscription.LatestInvoice.PaymentIntent == nil {
		data["payment"] = "unavailable"
	} else {
		data["payment"] = subscription.LatestInvoice.PaymentIntent.Status
	}

	// Check if latest_invoice is nil
	if subscription.LatestInvoice == nil {
		data["invoice"] = "unavailable"
	} else {
		data["invoice"] = subscription.LatestInvoice.Status
	}

	data["subscription"] = subscription.Status

	c.Response.Status = 200
	return c.RenderJSON(data)
}
