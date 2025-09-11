package ob

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/objectbox/objectbox-go/objectbox"
)

//go:generate go run github.com/objectbox/objectbox-go/cmd/objectbox-gogen

// Simple entity
type Person struct {
	ID   uint64 `objectbox:"id"`
	Name string
	Age  int
}

func SimpleExample() {
	tempDir, err := os.MkdirTemp("", "objectbox_simple")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize ObjectBox
	ob, err := objectbox.NewBuilder().Directory(filepath.Join(tempDir, "objectbox")).Build()
	if err != nil {
		panic(err)
	}
	defer ob.Close()

	// Get box for Person entities
	box := BoxForPerson(ob)

	// Create some people
	people := []*Person{
		{Name: "Alice", Age: 30},
		{Name: "Bob", Age: 25},
		{Name: "Charlie", Age: 35},
	}

	// Put (insert/update) all people
	ids, err := box.PutMany(people)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Inserted %d people with IDs: %v\n", len(ids), ids)

	// Query by age
	query := box.Query(Person_.Age.Between(25, 30))
	results, err := query.Find()
	query.Close()
	
	if err != nil {
		panic(err)
	}

	fmt.Printf("Found %d people aged 25-30:\n", len(results))
	for _, person := range results {
		fmt.Printf("- %s (age %d)\n", person.Name, person.Age)
	}

	// Get all people
	all, err := box.GetAll()
	if err != nil {
		panic(err)
	}

	fmt.Printf("All people in database:\n")
	for _, person := range all {
		fmt.Printf("- ID: %d, Name: %s, Age: %d\n", person.ID, person.Name, person.Age)
	}
}
