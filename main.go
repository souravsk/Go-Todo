package main

import (
	"context"
	"encoding/json" /* it is to encode the data that we get from mongodb */
	"log"      /* we need to this to see log if there is any error */
	"net/http" /* This is the main package that allow as to create a server in golang */
	"os"
	"os/signal"
	"strings" /* to perform string opreation with get post */
	"time"

	"github.com/go-chi/chi" /* this package help me to create route  */
	"github.com/go-chi/chi/middleware"
	"github.com/thedevsaddam/renderer" /* this is renderer that renders the web page for the todo list */

	/* library  that help to talk to the mongodb database */
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson" /* bson is database storage format that is used my mongoDB */
)

/* This are the varibales that we create so that we can use there propartices */
var render *renderer.Render
var db *mgo.Database

/* THis are the constent varible we don't want to make any change that why we created const */
const(
	hostName		string = "localhost:27017"
	dbName			string = "demo_todo"
	collectionName  string = "todo"
	port			string = ":9000"
)

type(
	todoModel struct{
		ID		bson.ObjectId `bson:"_id,omitempty"`
		Title	string	`bson:"title"`
		Completed bool  `bson:"completed"`
		CreatedAt time.Time `bson:"createAt"`
	}

	todo struct{
		ID		string `josn:"id"`
		Title 	string `josn:"title"`
		Completed bool `josn:"completed"`
		CreatedAt time.Time `json:"createAt"`
	}
)

/* This is called an init function it will excute only once and befor the main function */
func init(){
	render = renderer.New()
	sess, err:=mgo.Dial(hostName) /* we are creating a section  */
	checkErr(err) /* checking for error if any */
	sess.SetMode(mgo.Monotonic, true) /*  */
	db = sess.DB(dbName)
}

func homeHandler(w http.ResponseWriter, r *http.Request){
	err := render.Template(w, http.StatusOK, []string{"static/home.tpl"},nil)
	checkErr(err)
}

func fetchTodos(w http.ResponseWriter, r *http.Request){
	todos := []todoModel{}

	if err := db.C(collectionName).Find(bson.M{}).All(&todos); err != nil{
		render.JSON(w, http.StatusProcessing, renderer.M{
			"message":"Failed to fetch todos",
			"err":err,
		})
		return
	}
	todoList := []todo{}

	for _,t := range todos{
		todoList = append(todoList, todo{
			ID: t.ID.Hex(),
			Title: t.Title,
			Completed: t.Completed,
			CreatedAt: t.CreatedAt,
		})
	}
	render.JSON(w, http.StatusOK, renderer.M{
		"data":todoList,
	})
}

/* Here we are creating the data and storeing into mongodb */
func createTodo(w http.ResponseWriter, r *http.Request){
	var t todo

	if err:= json.NewDecoder(r.Body).Decode(&t); err != nil{
		render.JSON(w, http.StatusProcessing,err)
		return
	}
	/* checking if the var is not empty */
	if t.Title == ""{
		render.JSON(w, http.StatusBadRequest, renderer.M{
			"message":"The title is required",
		})
		return
	}

	/*here we are getting all the data  */
	tm := todoModel{
		ID: bson.NewObjectId(),
		Title: t.Title,
		Completed: false,
		CreatedAt: time.Now(),
	}
	/*  here we are send to mongodb to store */
	if err:= db.C(collectionName).Insert(tm); err != nil{
		render.JSON(w, http.StatusProcessing, renderer.M{
			"message":"Fail to save the data",
		})
		return
	}
	/* here print success messaage */
	render.JSON(w, http.StatusCreated, renderer.M{
		"message":"todo created successfully",
		"todo_id":tm.ID.Hex(),
	})
}

/* This function is for deleting the list iteam */
func deleteTodo(w http.ResponseWriter, r *http.Request){
	id := strings.TrimSpace(chi.URLParam(r,"id")) /* we created a variable to get the id. string is to remove space and chi will get the url and id  */

	/* here we check weather ID is valid or not */
	if !bson.IsObjectIdHex(id){
		render.JSON(w, http.StatusProcessing, renderer.M{
			"message":"This id is invalid",
		})
		return
	}

	/* here we will work with database or mongodb */
	if err:= db.C(collectionName).RemoveId(bson.IsObjectIdHex(id));err != nil{
		render.JSON(w, http.StatusProcessing, renderer.M{
			"message":"Failed to delete todo",
			"error":err,
		})
		return
	}

	/* Here we will send success full message for deleteting */
	render.JSON(w, http.StatusOK, renderer.M{
		"message":"todo Deleted Succesfully",
	})
}

/* This function is for updateing the todo list */
func updateTodo(w http.ResponseWriter, r *http.Request){
	id := strings.TrimSpace(chi.URLParam(r,"id"))

	if !bson.IsObjectIdHex(id){
		render.JSON(w, http.StatusBadRequest, renderer.M{
			"message":"The ID is invalid",
		})
		return
	}

	var t todo

	if err:= json.NewDecoder(r.Body).Decode(&t); err != nil{
		render.JSON(w, http.StatusProcessing, err)
		return
	}

	if t.Title == ""{
		render.JSON(w, http.StatusBadRequest, renderer.M{
			"message":"The title is field id required",
		})
		return
	}

	if err:= db.C(collectionName).
	Update(
		bson.M{"_id":bson.ObjectIdHex(id)},
		bson.M{"title":t.Title,"completed":t.Completed},
	); err != nil{
		render.JSON(w, http.StatusProcessing, renderer.M{
			"message":"Faile to update todo",
			"error":err,
		})
		return
	}
}

/* As the name say it is main function */
func main(){
	/* this is to stop the go channal or go server */
	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt)

	r := chi.NewRouter() /* Creating route with chi */
	r.Use(middleware.Logger)
	r.Get("/", homeHandler) /* This fuction will be called when we hit localhost */
	r.Mount("/todo", todoHandler()) /* This fuction will be called when we hit localhost/todo */

	/* Now we will create the server */
	srv := &http.Server{
		Addr: port, /* that we have created */
		Handler: r, /* is the on how handle the operation  */
		ReadTimeout: 60*time.Second, /* this are the time for the server to wait to get any kind of request */
		WriteTimeout: 60*time.Second,
		IdleTimeout: 60*time.Second,
	}
	go func ()  {
		log.Println("Listening on Port",port)
		if err:= srv.ListenAndServe();
		err != nil{
			log.Printf("listen:%s\n",err)
		}
	}()

	<-stop
	log.Println("Shutting Down The Server......")
	cxt, cancle:= context.WithTimeout(context.Background(),5*time.Second)
	srv.Shutdown(cxt)
	defer cancle()
		log.Println("Server Stopped")
}


func todoHandler() http.Handler{
	rg := chi.NewRouter()
	rg.Group(func(r chi.Router){ /* this are the operation we will do on todo list rg is group router */
		r.Get("/",fetchTodos)
		r.Post("/",createTodo)
		r.Put("/{id}",updateTodo)
		r.Delete("/{id}",deleteTodo)
	})
	return rg
}

/* we check if the err is not nil then we are printing the error */
func checkErr(err error){
	if err != nil{
		log.Fatal(err)
	}
}