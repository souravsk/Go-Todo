package main

import (
	"encodeing/json" /* it is to encode the data that we get from mongodb */
	"log" /* we need to this to see log if there is any error */
	"net/http" /* This is the main package that allow as to create a server in golang */
	"strings" /* to perform string opreation with get post */
	"time"
	"context"
	"os"
	"os/signal"

	"github.com/go-chi/chi" /* this package help me to create route  */
	"github.com/go-chi/chi/middleware"
	"github.com/thedevsaddam/renderer"  /* this is renderer that renders the web page for the todo list */

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
		CreatedAt time.Time `bson:"createAt"`
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
		todoList = append(todoList, todos{
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

/* As the name say it is main function */
func main(){
	/* this is to stop the go channal or go server */
	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt)

	r := chi.NewRouter() /* Creating route with chi */
	r.Use(middleware.Logger)
	r.Get("/", homeHandler()) /* This fuction will be called when we hit localhost */
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
	defer cancle(
		log.Println("Server Stopped")
	)
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