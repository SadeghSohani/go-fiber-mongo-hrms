package main

import (
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	_ "go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	_ "go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	_ "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	_ "go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/net/context"
	"log"
	"time"
)

type MongoInstance struct {
	Client *mongo.Client
	Db     *mongo.Database
}

var mg MongoInstance

const dbName = "fiber-hrms"
const mongoURI = "mongodb://127.0.0.1:27017/" + dbName

type Employee struct {
	ID     string  `json:"id,omitempty" bson:"_id,omitempty"`
	Name   string  `json:"name"`
	Salary float64 `json:"salary"`
	Age    float64 `json:"age"`
}

func Connect() error {
	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	db := client.Database(dbName)
	if err != nil {
		return err
	}
	mg = MongoInstance{
		client, db,
	}
	return nil
}
func main() {
	if err := Connect(); err != nil {
		log.Fatal(err)
	}
	app := fiber.New()

	app.Get("/employee", func(c *fiber.Ctx) error {
		query := bson.D{{}}
		cursor, err := mg.Db.Collection("employees").Find(c.Context(), query)
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}
		var employees = make([]Employee, 0)
		if err := cursor.All(c.Context(), &employees); err != nil {
			return c.Status(500).SendString(err.Error())
		}
		return c.JSON(employees)
	})

	app.Post("/employee", func(c *fiber.Ctx) error {
		collection := mg.Db.Collection("employees")
		employee := new(Employee)
		if err := c.BodyParser(employee); err != nil {
			return c.Status(500).SendString(err.Error())
		}
		employee.ID = ""
		insertionResult, err := collection.InsertOne(c.Context(), employee)
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}
		filter := bson.D{{Key: "_id", Value: insertionResult.InsertedID}}
		createdRecord := collection.FindOne(c.Context(), filter)
		createdEmployee := &Employee{}
		if err := createdRecord.Decode(createdEmployee); err != nil {
			return c.Status(500).SendString(err.Error())
		}
		return c.Status(201).JSON(createdEmployee)
	})

	app.Put("/employee/:id", func(c *fiber.Ctx) error {
		idParam := c.Params("id")
		employeeId, err := primitive.ObjectIDFromHex(idParam)
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}
		employee := new(Employee)
		if err := c.BodyParser(employee); err != nil {
			return c.Status(500).SendString(err.Error())
		}
		query := bson.D{{Key: "_id", Value: employeeId}}
		//update := bson.D{
		//	{Key: "name", Value: employee.Name},
		//	{Key: "salary", Value: employee.Salary},
		//	{Key: "age", Value: employee.Salary},
		//}
		update := bson.M{
			"$set": Employee{Name: employee.Name, Salary: employee.Salary, Age: employee.Age},
		}
		_err := mg.Db.Collection("employees").FindOneAndUpdate(c.Context(), query, update).Err()
		if _err != nil {
			if _err == mongo.ErrNoDocuments {
				return c.Status(404).SendString("Document not found!")
			}
			return c.Status(500).SendString(_err.Error())
		}
		employee.ID = idParam
		return c.Status(200).JSON(employee)
	})

	app.Delete("/employee/:id", func(c *fiber.Ctx) error {
		employeeID, err := primitive.ObjectIDFromHex(c.Params("id"))
		if err != nil {
			return c.SendStatus(400)
		}
		query := bson.D{{Key: "_id", Value: employeeID}}
		result, _err := mg.Db.Collection("employees").DeleteOne(c.Context(), &query)
		if _err != nil {
			return c.SendStatus(500)
		}
		if result.DeletedCount < 1 {
			return c.Status(404).SendString("Record not founded.")
		}
		return c.Status(200).SendString("Record deleted successfully.")
	})
	log.Fatal(app.Listen(":3000"))
}
