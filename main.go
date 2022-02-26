package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var schema = `CREATE TABLE public.recipe (
	description text NOT NULL,
	recipe_name character varying(100) NOT NULL,
	id serial NOT NULL
  );
  ALTER TABLE
	public.recipe
  ADD
	CONSTRAINT recipe_pkey PRIMARY KEY (id);
	
	CREATE TABLE public.ingredient (
		recipe_id integer NOT NULL,
		unit character varying(10) NOT NULL,
		quantity integer NOT NULL,
		ingredient_name character varying(20) NOT NULL,
		id serial NOT NULL,
		CONSTRAINT fk_recipe
      	FOREIGN KEY(recipe_id) 
	  	REFERENCES recipe(id)
	  );
	  ALTER TABLE
		public.ingredient
	  ADD
		CONSTRAINT ingredient_pkey PRIMARY KEY (id);`

type Recipe struct {
	Id          int
	Name        string
	Description string
	Ingredients []IngredientDTO
}

type RecipeDTO struct {
	Id          int
	Name        string `db:"recipe_name"`
	Description string
}

type IngredientDTO struct {
	Id       int
	Name     string `db:"ingredient_name"`
	Quantity int    `db:"quantity"`
	Unit     string `db:"unit"`
}

type RecipeDetails struct {
	Recipe      RecipeDTO
	Ingredients []IngredientDTO
}

func connectToDB() *sqlx.DB {
	db, err := sqlx.Connect("postgres", "user=postgres dbname=hellofresh password=mysecretpassword sslmode=disable")
	if err != nil {
		log.Fatalln(err)
	}
	return db
}

func getRecipe(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var db = connectToDB()
	recipe := RecipeDTO{}
	ingredient := []IngredientDTO{}
	recipeDetails := RecipeDetails{}
	err := db.Get(&recipe, "SELECT * FROM recipe where recipe.id = $1", id)
	db.Select(&ingredient, "SELECT id, ingredient_name, quantity, unit FROM ingredient where recipe_id = $1", id)
	if err != nil {
		w.WriteHeader(204)
		w.Write([]byte("No recipe found for given Id"))
	} else {
		recipeDetails.Recipe = recipe
		recipeDetails.Ingredients = ingredient
		recipe, _ := json.Marshal(recipeDetails)
		w.Header().Add("Content-Type", "aplication/json")
		w.WriteHeader(200)
		w.Write(recipe)
	}
}

func addRecipe(w http.ResponseWriter, r *http.Request) {
	var recipe Recipe
	json.NewDecoder(r.Body).Decode(&recipe)

	log.Println(r.Body)
	var db = connectToDB()
	tx := db.MustBegin()
	//res, err := tx.NamedExec("INSERT INTO recipe (recipe_name, description) VALUES (:name, :description) returning id", &Recipe{recipe.Id, recipe.Name, recipe.Description, recipe.Ingredients})
	res, err := tx.PrepareNamed("INSERT INTO recipe (recipe_name, description) VALUES (:name, :description) returning id")
	var lastInsertedId int
	err = res.Get(&lastInsertedId, recipe)
	for i := 0; i < len(recipe.Ingredients); i++ {
		query := "INSERT INTO ingredient (ingredient_name, quantity, unit, recipe_id) VALUES ($1, $2, $3, $4)"
		tx.Exec(query, recipe.Ingredients[i].Name, recipe.Ingredients[i].Quantity, recipe.Ingredients[i].Unit, lastInsertedId)
	}
	if err != nil {
		log.Fatalln(err)
		log.Println(res)
		w.Write([]byte("Failed while adding recipe"))
	} else {
		w.WriteHeader(201)
		w.Write([]byte("Recipe successfully added"))
	}
	tx.Commit()
}

func updateRecipe(w http.ResponseWriter, r *http.Request) {
	var recipe Recipe
	json.NewDecoder(r.Body).Decode(&recipe)

	var db = connectToDB()
	tx := db.MustBegin()
	query := "UPDATE recipe SET name = $1, description = $2 WHERE id = $4"
	tx.Exec(query, recipe.Name, recipe.Description, recipe.Id)
	tx.Commit()
	w.WriteHeader(200)
	w.Write([]byte("Recipe updated successfully"))
}

func deleteRecipe(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var db = connectToDB()
	db.Exec("DELETE FROM recipe where id = $1", id)

	w.WriteHeader(200)
	w.Write([]byte("Recipe deleteed successfully"))
}

type Server struct {
	Router *chi.Mux
}

func CreateNewServer() *Server {
	s := &Server{}
	s.Router = chi.NewRouter()
	return s
}

func (r *Server) MountHandlers() {
	r.Router.Use(middleware.Logger)

	//r.Router.Get("/recipes", getRecipes)
	r.Router.Get("/recipes/{id}", getRecipe)
	r.Router.Post("/recipes", addRecipe)
	r.Router.Put("/recipes", updateRecipe)
	r.Router.Delete("/recipes/{id}", deleteRecipe)

}

func main() {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Route("/recipes", func(r chi.Router) {
		//r.Get("/", getRecipes)
		r.Get("/{id}", getRecipe)
		r.Post("/", addRecipe)
		r.Put("/", updateRecipe)
		r.Delete("/{id}", deleteRecipe)
	})

	//database connection
	// db, err := sqlx.Connect("postgres", "user=postgres dbname=hellofresh password=mysecretpassword sslmode=disable")
	// if err != nil {
	// 	log.Fatalln(err)
	// }
	// db.MustExec(schema)

	http.ListenAndServe(":3000", r)
}
