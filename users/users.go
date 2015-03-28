// Handling of users (creation, management, etc) will
// be done in here
// Defines the user model and interactions/utilities for
// working with users

package users

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/njdup/serve/db"
	"github.com/njdup/serve/utils/security"
)

// The User struct defines the database fields associated
// with a registered user
type User struct {
	Id       bson.ObjectId `bson:"_id,omitempty" json:"-"`
	Inserted time.Time     `bson:"inserted" json"-"`

	Username     string `bson:"userName" json:"userName"`
	Firstname    string `bson:"firstName" json:"firstName"`
	Lastname     string `bson:"lastName" json:"lastName"`
	Phonenumber  string `bson:"phoneNumber" json:"phoneNumber`
	PasswordHash string `bson:"password" json:"-"`
}

var (
	CollectionName = "users" // Name of the collection in mongo
)

// Returns a string representation of the user object
func (user *User) ToString() string {
	return fmt.Sprintf(
		"User %s (%s): %s %s",
		user.Username,
		user.Phonenumber,
		user.Firstname,
		user.Lastname,
	)
}

// Inserts the receiver User into the database
// Returns an error if any are encountered, including
// validation errors
func (user *User) Save() error {
	if emptyFields := checkEmptyFields(user); len(emptyFields) != 0 {
		invalid := strings.Join(emptyFields, " ")
		return errors.New("The following fields cannot be empty: " + invalid)
	}

	insertQuery := func(col *mgo.Collection) error {
		nameCh := make(chan int)
		go checkExistence(col, bson.M{"userName": user.Username}, nameCh)

		phoneCh := make(chan int)
		go checkExistence(col, bson.M{"phoneNumber": user.Phonenumber}, phoneCh)

		if nameMatches := <-nameCh; nameMatches != 0 {
			return errors.New("A user with the given username already exists")
		}

		if phoneMatches := <-phoneCh; phoneMatches != 0 {
			return errors.New("A user with the given phone number already exists")
		}

		user.Inserted = time.Now()
		return col.Insert(user) // Inserts the user, returning nil or an error
	}

	return db.ExecWithCol(CollectionName, insertQuery)
}

func checkExistence(col *mgo.Collection, query bson.M, ch chan int) {
	count, err := col.Find(query).Limit(1).Count()
	if err != nil {
		ch <- -1 // TODO: Is there a better way to handle an error here?
		return
	}
	ch <- count
}

// Stores the given password for the user after hashing
// Returns the error encountered while hashing the password if applicable,
// otherwise nil is returned
func (user *User) SetPassword(password string) error {
	if !security.PasswordPolicy.PasswordValid(password) {
		return errors.New("Given password is not acceptable")
	}
	var err error
	user.PasswordHash, err = security.HashPassword(password)
	return err
}

// Checks whether the given password matches the password for the user
func (user *User) PasswordsMatch(givenPassword string) bool {
	return security.ConfirmPassword(user.PasswordHash, givenPassword)
}

// Checks whether the required fields of a user object are set
// Returns a splice of all required fields that are empty
func checkEmptyFields(user *User) []string {
	result := make([]string, 0)

	if user.Username == "" {
		result = append(result, "Username")
	}

	if user.Phonenumber == "" {
		result = append(result, "Phonenumber")
	}

	return result
}
