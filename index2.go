package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	pag "github.com/gobeam/mongo-go-pagination"
	"github.com/labstack/echo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var connect *mongo.Client = nil

//User infomation for rookie
type User struct {
	ID        primitive.ObjectID `json:"id"     bson:"_id,omitempty"`
	Name      string             `json:"name"   bson:"name,omitempty"`
	Avatar    string             `json:"avatar" bson:"avatar,omitempty"`
	Age       int                `json:"age"    bson:"age"`
	Birth     int                `json:"birth"    bson:"birth"`
	Note      string             `json:"note"   bson:"note,omitempty"`
	Email     string             `json:"email"  bson:"email,omitempty"`
	Avatype   string             `json:"avatype"  bson:"avatype,omitempty"`
	Avaname   string             `json:"avaname"  bson:"avaname,omitempty"`
	Fileid    primitive.ObjectID
	Createdat time.Time `json: "create_at" bson:"create_at,omitempty"`
	Updateat  time.Time `json: "update_at" bson:"update_at,omitempty"`
}

type Filestore struct {
	ID primitive.ObjectID `json:"id" bson:"_id,omitempty"`
}

func yallo(c echo.Context) error {
	return c.String(http.StatusOK, "Yallooooo form web side!!")
}

//test function
func getUser(c echo.Context) error {

	contentType := c.Request().Header.Get("Content-Type")
	fmt.Println(contentType)
	uName := c.QueryParam("name")
	uAva := c.QueryParam("ava")
	// uAge := c.QueryParam("age")
	// uNote := c.QueryParam("note")
	// uEmail := c.QueryParam("email")

	dataType := c.Param("data")
	return c.String(http.StatusOK, fmt.Sprintf("Your name is: %s\nYour avatar is:%s\nYour age is:%s\n", uName, uAva))
	if dataType == "string" {
		return c.String(http.StatusOK, fmt.Sprintf("Your name is: %s\nYour avatar is:%s\nYour age is:%s\n", uName, uAva))
	}

	if dataType == "json" {
		return c.JSON(http.StatusOK, map[string]string{
			"name":   uName,
			"avatar": uAva,
			// "age":    uAge,
			// "note":   uNote,
			// "email":  uEmail,
		})
	}
	return c.JSON(http.StatusBadRequest, map[string]string{
		"error": "you need to lets us know if u want json or string",
	})
}

//POST Fucntion
func addUser(c echo.Context) error {

	defer c.Request().Body.Close()

	currTime := time.Now()
	yearCal := currTime.Year()

	user := User{}

	a := c.FormValue("file")  // data input
	p, _ := c.FormFile("pic") // picture file

	extension := filepath.Ext(p.Filename)
	fName := p.Filename
	fileName := fName[0 : len(fName)-len(extension)]

	fmt.Println("Extension:", extension) //check file type ****
	fmt.Println("File name:", fileName)  //check file name ****
	if extension != ".png" && extension != ".jpg" {
		return c.JSON(http.StatusUnauthorized, "Extension fiel should be jpg and png!!!")
	}
	// fmt.Println([]byte(a))  check type data ****

	//-------------------use only read form body------------------
	// b, err := ioutil.ReadAll(c.Request().Body)
	// if err != nil {
	// 	log.Printf("Failed reading the request body: %s", err)
	// 	return c.String(http.StatusInternalServerError, "")
	// }
	//------------------------------------------------------------

	err := json.Unmarshal([]byte(a), &user)
	user.Createdat = currTime
	user.Updateat = currTime
	user.Avaname = fileName
	user.Avatype = extension
	user.Birth = yearCal - user.Age
	yearBirth := user.Birth
	fmt.Println("Here is your year of birth:", yearBirth)
	// log.Printf("Here: \n %#v", user.Birth) check age

	if err != nil {
		log.Printf("Failed unmarshal in addUser: %s", err)
		return c.String(http.StatusInternalServerError, "")
	}

	if user.Name != "" && user.Avatar != "" && user.Age > 1 && user.Email != "" && user.Age < 100 {
		log.Printf("Here: /n %#v", user)
		db := connect.Database("customer")
		collection := db.Collection("rookie")
		insertUser, errs := collection.InsertOne(context.TODO(), user)
		if errs != nil {
			log.Fatal(errs)
		}
		fmt.Println("Inserted a single document: ", insertUser.InsertedID)

		return c.String(http.StatusCreated, "we got your name!!")
	} else if user.Age > 100 {
		return c.String(http.StatusUnauthorized, "Ur age is over 100 year or less than 1 year")
	}
	return c.String(http.StatusOK, "Please fill all infomation!!")
}

func addFile(c echo.Context) error {
	fmt.Println("hello")
	file, err := c.FormFile("file")
	if err != nil {
		log.Fatal(err)
		return err
	}
	fmt.Println(file.Filename)
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, src); err != nil {
		return err
	}
	// fmt.Println(buf.Bytes())
	bucket, _ := gridfs.NewBucket(
		connect.Database("customer"),
	)
	fileId := primitive.NewObjectID()
	uploadStream, err := bucket.OpenUploadStreamWithID(fileId, file.Filename)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer uploadStream.Close()
	fileSize, err := uploadStream.Write(buf.Bytes())
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	fileResult := &Filestore{
		ID: fileId,
	}
	log.Printf("Write file to DB was successful. File size: %d \n", fileSize)
	return c.JSON(http.StatusOK, fileResult)

}

//GET ALL Function
func getAlluser(c echo.Context) error {
	info := make([]map[string]interface{}, 0)

	db := connect.Database("customer")
	col := db.Collection("rookie")
	cur, err := col.Find(context.TODO(), bson.D{{}})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("check")
	if err = cur.All(context.TODO(), &info); err != nil {
		// fmt.Println("check2")
		log.Fatal(err)
	}
	// fmt.Println(">==========================TEST For Done========================<")
	// fmt.Println("Get all user")
	// fmt.Println(info[0])
	// fmt.Println(">==========================INFO1========================<")
	// fmt.Println(info[1])
	return c.JSON(http.StatusOK, info)
}

//GET ONE Function
func getOne(c echo.Context) error {

	infoOne := make(map[string]interface{}, 0)

	dataID := c.Param("one_id")
	oid, _ := primitive.ObjectIDFromHex(dataID)

	db := connect.Database("customer")
	col := db.Collection("rookie")
	err := col.FindOne(context.TODO(), bson.M{"_id": oid}).Decode(&infoOne)
	if err != nil {
		fmt.Println("Check")
		return c.JSON(http.StatusNotFound, "Not Found User ID")

	}
	return c.JSON(http.StatusOK, infoOne)
}

//GET LIMIT Function
func getLim(c echo.Context) error {
	info := make([]map[string]interface{}, 0)
	limit := c.QueryParam("limit")
	page := c.QueryParam("page")

	liint, _ := strconv.ParseInt(limit, 10, 64)
	lipag, _ := strconv.ParseInt(page, 10, 64)

	filter := bson.M{}
	var limits int64 = liint
	var pages int64 = lipag

	projection := bson.D{
		{"name", 1},
		{"avatar", 1},
		{"age", 1},
		{"birth", 1},
		{"note", 1},
		{"email", 1},
		{"avatype", 1},
		{"avaname", 1},
		{"Fileid", 1},
		{"Createdat", 1},
		{"Updateat", 1},
	}
	db := connect.Database("customer")
	col := db.Collection("rookie")
	cur, err := col.Find(context.TODO(), bson.D{{}})
	if err != nil {
		log.Fatal(err)
	}
	if err = cur.All(context.TODO(), &info); err != nil {
		log.Fatal(err)
	}
	paginatedData, err := pag.New(col).Limit(limits).Page(pages).Sort("price", -1).Select(projection).Filter(filter).Find()
	if err != nil {
		fmt.Println("check2")
		panic(err)
	}
	var lists []User
	for _, raw := range paginatedData.Data {
		var product *User
		if marshallErr := bson.Unmarshal(raw, &product); marshallErr == nil {
			lists = append(lists, *product)
		}

	}
	// print ProductList
	fmt.Printf("Norm Find Data: %+v\n", lists)

	// print pagination data
	fmt.Printf("Normal find pagination info: %+v\n", paginatedData.Pagination)
	return c.JSON(http.StatusOK, lists)
}

//UPDATE ONE Function
func upDate(c echo.Context) error {

	// updateinfo := make(map[string]interface{})
	// updateinfo["name"] = user.Name
	currTime := time.Now()
	db := connect.Database("customer")
	col := db.Collection("rookie")
	dataID := c.Param("user_id")
	user := User{}
	a := c.FormValue("file")

	user.Updateat = currTime

	err := json.Unmarshal([]byte(a), &user)

	if user.Note == "clear" {
		user.Note = ""
	}
	update := bson.M{
		"$set": bson.M{
			"name":     user.Name,
			"age":      user.Age,
			"note":     user.Note,
			"updateat": user.Updateat,
		},
	}

	rid, _ := primitive.ObjectIDFromHex(dataID)
	// fmt.Println("Here id ", rid)
	if err != nil {
		log.Printf("Failed unmarshal in addUser: %s", err)
		return c.String(http.StatusInternalServerError, "")
	}

	// cur, err := col.Find(context.TODO(), bson.M{})
	// if err != nil {
	// 	log.Fatal(cur)
	// 	return c.String(http.StatusUnauthorized, "Can't find user ID in database")
	// }
	// fmt.Println(cur)
	// newresult, err := col.UpdateOne(context.TODO(), bson.M{"_id": rid}, bson.D{"$set", bson.D{{
	// 	"name": user.Name,
	// }}})

	newresult, err := col.UpdateOne(context.TODO(), bson.M{"_id": rid}, update)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(newresult)
	return c.JSON(http.StatusOK, update)
}

//DELETE Function
func delOne(c echo.Context) error {
	dataID := c.Param("del_id")
	dId, _ := primitive.ObjectIDFromHex(dataID)

	db := connect.Database("customer")
	col := db.Collection("rookie")
	resualtDel, err := col.DeleteOne(context.TODO(), bson.M{"_id": dId})
	if err != nil {
		log.Fatal(err)

	}
	fmt.Printf("DeleteOne function has been activated %v \n", resualtDel.DeletedCount)
	if resualtDel.DeletedCount == 0 {
		return c.JSON(http.StatusNotFound, "The User ID is not in Database or not matching")
	}

	return c.JSON(http.StatusOK, map[string]string{
		"success": "true",
	})
}

func main() {

	conN()
	currt := time.Now()
	fmt.Println("Welcome to server", currt.Year())
	e := echo.New()
	e.GET("/", yallo)
	e.GET("/user/", getUser)
	e.GET("/get", getAlluser)
	e.GET("/getone/:one_id", getOne)
	e.GET("/limit/", getLim)
	e.POST("/cuser", addUser)
	e.POST("/fuser", addFile)
	e.PUT("/userid/:user_id", upDate)
	e.DELETE("/delone/:del_id", delOne)
	e.Start(":8000")
}

func conN() {
	c, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal("Error : %v", err)
	}
	connect = c
	fmt.Println("connect to Mongodb")
}
