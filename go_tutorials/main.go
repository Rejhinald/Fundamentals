package main
import "fmt"

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

	 fmt.Printf("Welcome to %v\n", cinemaName)
	 fmt.Printf("We have a total of %v movies and %v movies still available!\n", cinemaList, cinemaListAvailable)
	 fmt.Println("Get your movies now!")

  // in go, we should always declare the variable type
  
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

	// logic for renting movies (remaining movies - user rented movies)
	cinemaListAvailable = cinemaListAvailable - userRent
		// adding elements to the array (first name and last name)
	// old command for arrays: rents[0] = userFirstName + " " + userLastName
	rents = append(rents, userFirstName + " " + userLastName)
	// new command for slices
	// using this command will tidy up the array list (automatically expands when adding a new element)




	{!} // this is a block of code for the array example
	// fmt.Printf("The Whole array: %v\n", rents)
	// fmt.Printf("The first value in the array: %v\n", rents[0])
	// fmt.Printf("Array type %T\n", rents)
	// fmt.Printf("Array length %v\n", len(rents))
	{!} // this is a block of code for the array example

	fmt.Printf("Thank you %v %v for renting %v movies, you will receive a confirmation email through your email: %v, until next time! \n", userFirstName, userLastName, userRent, userEmail)
	fmt.Printf("We have %v movies left in our inventory!\n", cinemaListAvailable)

	// next is to save the user data & movies rented to a file/list
	/// Loops in Go 1:11:30 https://www.youtube.com/watch?v=yyUHQIec83I
}