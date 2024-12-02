package main
import (
"fmt"
"strings"	
)


func main() {
	// Variables (:= is a short declaration operator for variables)
	cinemaName := "Arwin's movies"
	var cinemaListAvailable uint = 20
	// Constants
	const cinemaList uint = 20
	// rents is the total amount people can rent with a string array of 20
	var rents = []string{}
	// How would we know what size of the array we need?
	// We can use slice because slice is an abstraction of an arrray
	// Slices are more flexible than arrays since they are resized when needed

// calling the greetings function
	greetings(cinemaName, cinemaList, cinemaListAvailable)

  // in go, we should always declare the variable type
  // for is a loop in go
  // it will keep asking the user for their name and how many movies they want to rent until the user stops the program
	 for {
		var userRent uint
		var userFirstName string
		var userLastName string
		var userEmail string
		// ask user for their name
		// User Input (Input and Output functionality)
		// & is a pointer to the variable (memory address)
	   fmt.Println("What is your first name?")
	   fmt.Scan(&userFirstName)
	   fmt.Println("What is your last name?")
	   fmt.Scan(&userLastName)
	   fmt.Println("What is your email?")
	   fmt.Scan(&userEmail)
	   fmt.Printf("Hello %v %v, how many movies would you like to rent?\n", userFirstName, userLastName)
	   fmt.Scan(&userRent)
   
	   isValidName := len(userFirstName) >= 2 && len(userLastName) >= 2
	   isValidEmail := strings.Contains(userEmail, "@") && strings.Contains(userEmail, ".")
	   isValidRent := userRent > 0 && userRent <= cinemaListAvailable

	   // adding a condition statement where the user can't rent more than the available movies
	   if isValidName && isValidEmail && isValidRent {	
	   // logic for renting movies (remaining movies - user rented movies)
	   cinemaListAvailable = cinemaListAvailable - userRent
		   // adding elements to the array (first name and last name)
	   // old command for arrays: rents[0] = userFirstName + " " + userLastName
	   rents = append(rents, userFirstName + " " + userLastName)
	   // new command for slices
	   // using this command will tidy up the array list (automatically expands when adding a new element)
   
	   // this is a block of code for the array example
	   // fmt.Printf("The Whole array: %v\n", rents)
	   // fmt.Printf("The first value in the array: %v\n", rents[0])
	   // fmt.Printf("Array type %T\n", rents)
	   // fmt.Printf("Array length %v\n", len(rents))
	   // this is a block of code for the array example
   
	   fmt.Printf("Thank you %v %v for renting %v movies, you will receive a confirmation email through your email: %v, until next time! \n", userFirstName, userLastName, userRent, userEmail)
	   fmt.Printf("We have %v movies left in our inventory!\n", cinemaListAvailable)

	   // this block of code is to get the first name in the slice/array
	   firstNames := []string{}
	   for _, rent := range rents {
		   var names = strings.Fields(rent) 
		   firstNames = append(firstNames, names[0])
	   }
	   
	   fmt.Printf("These are all the people who rented movies: %v\n", firstNames)
	   // next is to save the user data & movies rented to a file/list

	   if cinemaListAvailable == 0 {
		// ending the program if there are no more movies left
		   fmt.Println("We are out of movies! Thank you for renting with us!")
		}
		} else {
			if !isValidName {
				fmt.Println("Your first name or last name is too short, please try again!")
			} 
			if !isValidEmail {
				fmt.Println("Your email is invalid, please try again!")
			}
			if !isValidRent {
				fmt.Println("You can't rent 0 or more than the available movies, please try again!")
			}
		}
	 }   
}
// moving the greetings to a function

func greetings(cinemaName string, cinemaList uint, cinemaListAvailable uint) {
	fmt.Printf("Welcome to %v\n", cinemaName)
	fmt.Printf("We have a total of %v movies and %v movies still available!\n", cinemaList, cinemaListAvailable)
	fmt.Println("Get your movies now!")
}

// i need to move the user input to a function 2:10:00
// i need to move the user input validation to a function


// By moving the functions into a separate function, we can make the code more readable and easier to maintain //
// We can also reuse the functions in other parts of the code //