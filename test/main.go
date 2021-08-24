package main

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-pg/pg"
	"github.com/golang-jwt/jwt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Author struct {
	AuthorId   int64  `pg:"author_id"`
	AuthorName string `pg:"author_name"`
}

type Genre struct {
	GenreId int64  `pg:"genre_id"`
	Genre   string `pg:"genre"`
}

type bookTokens struct {
	Id			int64		`pg:"id"`
	BookId        int64     `pg:"book_id"`
	Token          string   `pg:"token"`
	CreatedAt	time.Time	`pg:"created_at"`
}

type Reader struct {
	ReaderId         int64     `pg:"reader_id"`
	Name             string    `pg:"name"`
	BirthDate        time.Time `pg:"birth_date"`
	RegistrationDate time.Time `pg:"registration_date"`
}

type Book struct {
	BookId        int64     `pg:"book_id"`
	Name          string    `pg:"name"`
	AuthorId      int64     `pg:"author_id"`
	GenreId       int64     `pg:"genre_id"`
	CurrentReader int64     `pg:"current_reader"`
	ReleaseDate   time.Time `pg:"release_date"`
	BookFilepath string `pg:"book_filepath"`
	ImageFilepath string `pg:"image_filepath"`
}

type RentalHistory struct {
	RentalId   int64     `pg:"rental_id"`
	BookId     int64     `pg:"book_id"`
	ReaderId   int64     `pg:"reader_id"`
	RentalDate time.Time `pg:"rental_date"`
	ReturnDate time.Time `pg:"return_date"`
}

type TimeIntervalsForHistory struct {
	RentalDateFrom time.Time
	RentalDateTo   time.Time
	ReturnDateFrom time.Time
	ReturnDateTo   time.Time
}
type bookSearch struct {
	BookId        int64     `pg:"book_id"`
	Book          string    `pg:"book"`
	Author        string    `pg:"author"`
	ReleaseDate   time.Time `pg:"release_date"`
	Genre         string    `pg:"genre"`
	CurrentReader int64     `pg:"current_reader"`
	ImageFilepath string `pg:"image_filepath"`
}

type searchParams struct {
	OrderBy string
	Offset  int `pg:"offset"`
	Status  string
	Author string `pg:"author.author_name"`
}

type Users struct {
	Id         int64      `pg:"id"`
	Name        string    `pg:"name"`
	Password    string    `pg:"password"`
	Role 		string 		`pg:"role"`
}
type Roles struct {
	Id   int64  `pg:"id"`
	Role string `pg:"role"`
}
type jwtRefreshClaims struct {
	Id int64
	jwt.StandardClaims
}

type jwtAccessClaims struct {
	Id int64
	User string
	Role string
	Pop string
	jwt.StandardClaims
}

var db *pg.DB
var jwtKey = []byte("testKey")

func main() {

	db = pg.Connect(&pg.Options{
		User:     "postgres",
		Password: "admin",
		Database: "library",
	})
	defer db.Close()

	r := gin.Default()

	r.POST("refresh", validateRefreshToken)
	r.POST("login", login)

	r.Use(verifyAccessToken)

	authorsApi := r.Group("api/authors")
	authorsApi.GET("", allAuthors)
	authorsApi.POST("", createAuthor)
	authorsApi.DELETE("", deleteAuthor)
	authorsApi.PUT("", updateAuthor)

	genreApi := r.Group("api/genres")
	genreApi.GET("", allGenres)
	genreApi.POST("", createGenre)
	genreApi.DELETE("", deleteGenre)
	genreApi.PUT("", updateGenre)

	readerApi := r.Group("api/readers")
	readerApi.GET("", allReaders)
	readerApi.POST("", createReader)
	readerApi.DELETE("", deleteReader)
	readerApi.PUT("", updateReader)

	bookApi := r.Group("api/books")
	bookApi.GET("", showBooks)
	bookApi.POST("", createBook)
	bookApi.DELETE("", deleteBook)
	bookApi.PUT("", updateBook)

	userApi := r.Group("api/users")
	userApi.GET("*id", getUser)
	userApi.POST("", createUsers)
	userApi.DELETE("", deleteUser)
	userApi.PUT("", changePassword)

	roleApi := r.Group("api/roles")
	roleApi.GET("*id", getRoles)
	roleApi.POST("", createRole)
	roleApi.DELETE("", deleteRole)
	roleApi.PUT("", changeRole)

	//r.POST("/api/rentbook", rentABook)
	r.POST("/api/returnbook", returnBook)
	r.POST("/api/rentalhistory", showHistory)
	r.POST("/save",saveFile)
	r.GET("logout", logout)
	r.POST("take-book", takeBook)
	r.POST("load-book", loadBook)

	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}

func loadBook(c *gin.Context){
	var bookToken *bookTokens
	err := c.Bind(&bookToken)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	_, err = db.QueryOne(bookToken, `SELECT * FROM book_load_tokens WHERE token = ?`, bookToken.Token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	timeDifference := time.Now().Sub(bookToken.CreatedAt)
	if timeDifference.Hours() > 1 {
		c.JSON(http.StatusNotFound, gin.H{
			"Время токена истекло" : "",
		})
		return
	}
	bookPath, err := queryLoadBook(bookToken.Token)
	fmt.Println(timeDifference)
	data, err := ioutil.ReadFile(bookPath)
	fmt.Println(bookPath)
	if err != nil {
		fmt.Println("File reading error", err)
		return
	}
	fmt.Println(c.Keys["id"])
	readerId := c.Keys["id"].(int64)
	fmt.Println(readerId)

	err = rentABook(readerId,bookToken.BookId)
	if err != nil {
		fmt.Println("error", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"байты книги" : data,
	})
	//if err := os.WriteFile("file.pdf", data, 0666); err != nil { проверка файла
	//	log.Fatal(err)
	//}
}
func queryLoadBook(token string) (bookPath string,err error) {
	_, err = db.QueryOne(&bookPath, `SELECT book_filepath FROM book INNER JOIN book_load_tokens bl on bl.book_id = book.book_id WHERE token = ?`, token)
	if err != nil {
		fmt.Println("finduser err",err.Error())
		return "", err
	}
	return bookPath, err
}

func takeBook(c *gin.Context) {
	if c.Keys["role"] != "librarian"{
		c.JSON(http.StatusBadRequest, gin.H{"Недостаточно прав": ""})
		return}
	var loadBooks *bookTokens
	err := c.Bind(&loadBooks)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	loadBooks.Token, err = generateBookToken()
	if err != nil {
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": err.Error(),
	})
	return
	}
	_, err = db.QueryOne(loadBooks, `INSERT INTO book_load_tokens (token, book_id) VALUES (?token,?book_id) RETURNING *`, loadBooks)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"токен для закгрузки" : loadBooks.Token,
	})
}

func logout(c *gin.Context) {
	var id string
	fmt.Println(c.Keys["id"])
	_, err := db.QueryOne(&id, `DELETE FROM sessions WHERE user_id = ? RETURNING user_id`, c.Keys["id"])
	if err == nil {
		c.JSON(http.StatusOK, gin.H{
			id : "удален",
		})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{
		"error msg": err.Error(),
	})
}

func validateRefreshToken(c *gin.Context) {
	var RefreshPar struct {
		RefreshToken string `pg:"refresh_token"`
	}
	err := c.Bind(&RefreshPar)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	fmt.Println(RefreshPar)
	fmt.Println(RefreshPar.RefreshToken)
	user, err := findSession(RefreshPar.RefreshToken)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	accessToken, err := generateAccessToken(*user)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"accessToken": accessToken,
		"refreshToken": RefreshPar.RefreshToken,
	})
}

func findSession(refreshToken string) (*Users, error) {
	var user Users
	_, err := db.QueryOne(&user, `SELECT user_roles.user_id AS id, name, password, role FROM user_roles INNER JOIN roles r on r.id = user_roles.role_id INNER JOIN users u on u.id = user_roles.user_id INNER JOIN sessions s on s.user_id = user_roles.user_id WHERE refresh_token = ?`, refreshToken)
	if err != nil {
		fmt.Println("finduser err",err)
		return nil, err
	}
	fmt.Println(user)
	return &user, nil
}


func saveFile(c *gin.Context) {
	type loginPar struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	var jsonAndFile struct {
		login loginPar
		fileData *multipart.FileHeader
	}

	err := c.Bind(&jsonAndFile.login)

	//
	//err = c.ShouldBindJSON(&loginPar)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	jsonAndFile.fileData, err = c.FormFile("file")
	// The file cannot be received.
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})
		return
	}

	filePath := "resources/" +jsonAndFile.fileData.Filename
	//// File saved successfully. Return proper result
	//c.JSON(http.StatusOK, gin.H{
	//	"message": "Your file has been successfully uploaded.",
	//	"json":loginPar,
	//})
	// Retrieve file information
	//extension := filepath.Ext(file.Filename)
	// Generate random file name for the new uploaded file so it doesn't override the old file with same name
	//newFileName := uuid.New().String() + extension

	// The file is received, so let's save it
	if err := c.SaveUploadedFile(jsonAndFile.fileData, filePath); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message":err,
		})
		return
	}
	// File saved successfully. Return proper result
	c.JSON(http.StatusOK, gin.H{
		"message": "Your file has been successfully uploaded.",
		"json":jsonAndFile.fileData.Filename,
		"12313":jsonAndFile.login,
	})
}

func verifyAccessToken(c *gin.Context) {

	authValue := c.GetHeader("Authorization")
	arr := strings.Split(authValue, " ")
	//fmt.Println(arr,"arr")
	if len(arr) != 2 {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{})
		return
	}
	//authType := strings.Trim(arr[0], "\n\r\t")
	//fmt.Println(authType,"authtype")
	//if strings.ToLower(authType) != strings.ToLower("Bearer") {
	//	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{})
	//	return
	//}
	token := arr[1]
	//fmt.Println(token,"token")
	user, err := validateToken(token)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"Ошибка": err.Error()})
		return
	}
	if user.Name == ""{
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"рефреш токен не авторизирует": ""})
		return
	}
	c.Set("id", user.Id)
	c.Set("username", user.Name)
	c.Set("role", user.Role)
	//c.Writer.Header().Set("Authorization", "Bearer "+token)
	fmt.Println(c.Keys["username"])
	c.Next()
}

func login(c *gin.Context) {

	var loginPar struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	err := c.ShouldBindJSON(&loginPar)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "incorrect parameters",
		})
		return
	}
	user, err := findUser(&Users{
		Name: loginPar.Username,
		Password: loginPar.Password,
	})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("Неправильный логин или пароль"),
		})
		return
	}
	fmt.Println(user)
	accessToken, err := generateAccessToken(*user)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": err.Error(),
		})
		return
	}
	refreshToken, err := generateRefreshToken(*user)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": err.Error(),
		})
		return
	}
	err = startSession(user.Id,refreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"accessToken": accessToken,
		"refreshToken": refreshToken,
	})
}

func generateRefreshToken(user Users) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, jwtRefreshClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour * 24).Unix(),
		},
		Id: user.Id,
	})
	fmt.Println(token.Claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}
func generateBookToken() (string,error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.StandardClaims{
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	})
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func generateAccessToken(user Users) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, jwtAccessClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
		},
		User: user.Name,
		Role: user.Role,
		Id: user.Id,
	})
	fmt.Println(token.Claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func validateToken(tokenString string) (*Users, error) {
	var claims jwtAccessClaims
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
		_, ok := token.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtKey, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return &Users{
		Name: claims.User,
		Role: claims.Role,
		Id: claims.Id,
	}, nil
}

func findUser(user *Users) (*Users, error) {
	_, err := db.QueryOne(user, `SELECT user_id AS id, name, password, role FROM user_roles INNER JOIN roles r on r.id = user_roles.role_id INNER JOIN users u on u.id = user_roles.user_id WHERE name = ? AND password = ?`, user.Name,user.Password)
	if err != nil {
		fmt.Println("finduser err",err)
		return nil, err
	}
	fmt.Println(user)
	return user, nil
}

func startSession (id int64, token string) error {
	_, err := db.Exec(`INSERT INTO sessions (user_id, refresh_token) values (?,?)`, id,token)
	if err != nil {
		return err
	}
	return nil
}

func changeRole(c *gin.Context) {

	var role *Roles
	err := c.Bind(&role)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error msg": err.Error(),
		})
		return
	}
	_, err = db.QueryOne(role, `UPDATE roles SET role = (?role) WHERE id = (?id) RETURNING *`, role)
	if err == nil {
		c.JSON(http.StatusOK, gin.H{
			role.Role: "роль изменена",
		})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{
		"error msg": err.Error(),
	})

}

func deleteRole(c *gin.Context) {

	var role *Roles
	err := c.Bind(&role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error msg": err.Error()})
		return
	}

	_, err = db.QueryOne(role, `DELETE FROM roles WHERE id = ? RETURNING *`, role.Id)
	if err == nil {
		c.JSON(http.StatusOK, gin.H{
			role.Role : "удален",
		})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{
		"error msg": err.Error(),
	})
}

func createRole(c *gin.Context) {

	var role *Roles
	err := c.Bind(&role)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error msg": err.Error(),
		})
		return
	}
	_, err = db.QueryOne(role, `
		INSERT INTO roles (role) VALUES (?role) RETURNING *`, role)
	if err == nil {
		c.JSON(http.StatusOK, gin.H{
			role.Role : "Роль успешно добавлена",
		})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{
		"error msg": err.Error(),
	})

}

func getRoles(c *gin.Context) {
	var role []Roles
	id := c.Param("id")
	id = strings.ReplaceAll(id, "/", "")
	if id != "" {
		_, err := db.Query(&role, `SELECT * FROM roles WHERE id = (?)`, id)
		if err == nil {
			c.JSON(http.StatusOK, role)
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error msg": err.Error(),
		})
		return
	}
	//err := c.Bind(&role)
	//if err != nil {
	//	c.JSON(http.StatusBadRequest, gin.H{"error msg": err.Error()})
	//	return
	//}
	_, err := db.Query(&role, `SELECT * FROM roles`)
	if err == nil {
		c.JSON(http.StatusOK, role)
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{
		"error msg": err.Error(),
	})

}

func changePassword(c *gin.Context) {

	var user *Users
	err := c.Bind(&user)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error msg": err.Error(),
		})
		return
	}
	_, err = db.QueryOne(user, `UPDATE users SET password = (?password) WHERE id = (?id) RETURNING *`, user)
	if err == nil {
		c.JSON(http.StatusOK, gin.H{
			user.Name: "пароль изменен",
		})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{
		"error msg": err.Error(),
	})
}

func deleteUser(c *gin.Context) {

	var user *Users
	err := c.Bind(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error msg": err.Error()})
		return
	}
	if user.Id == 1{
		c.JSON(http.StatusBadRequest, gin.H{
			"Невозможно удалить данного пользователя":"",
		})
		return
	}
	_, err = db.QueryOne(user, `DELETE FROM users WHERE id = ? RETURNING *`, user.Id)
	if err == nil {
		c.JSON(http.StatusOK, gin.H{
			user.Name : "удален",
		})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{
		"error msg": err.Error(),
	})
}

func createUsers(c *gin.Context) {

		var user *Users
		err := c.Bind(&user)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error msg": err.Error(),
			})
			return
		}
		_, err = db.QueryOne(user, `
		INSERT INTO users (name, password) VALUES (?name,?password) RETURNING *`, user)
		if err == nil {
			c.JSON(http.StatusOK, gin.H{
				user.Name : "успешно добавлен",
			})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error msg": err.Error(),
		})
}

func getUser(c *gin.Context) {

		var user []Users
	id := c.Param("id")
	fmt.Println(id)
	id = strings.ReplaceAll(id, "/", "")
	if id != "" {
		_, err := db.Query(&user, `SELECT * FROM users WHERE id = (?)`, id)
		if err == nil {
			c.JSON(http.StatusOK, user)
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error msg": err.Error(),
		})
		return
	}
		_, err := db.Query(&user, `SELECT * FROM users`)
		if err == nil {
			c.JSON(http.StatusOK, user)
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error msg": err.Error(),
		})
	}


func BasicAuth() gin.HandlerFunc {

	var user []Users
	_, err := db.Query(&user, `SELECT * FROM users`)

	if err != nil {
		fmt.Println(err)
	}
	m:= make(map[string]string)
	for _, v:= range user{
		m[v.Name] = v.Password
	}
	fmt.Println(m)
	return gin.BasicAuth(m)
}


func showHistory(c *gin.Context) {

	var history []RentalHistory
	var interval *TimeIntervalsForHistory
	err := c.Bind(&interval)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error msg": err.Error()})
		return
	}

	_, err = db.Query(&history, `SELECT * FROM rental_history WHERE rental_date >= (?) AND rental_date <= (?)
AND return_date >= (?) AND return_date <= (?)`, interval.RentalDateFrom, interval.RentalDateTo, interval.ReturnDateFrom, interval.ReturnDateTo)
	if err == nil {
		c.JSON(200, gin.H{
			"result": history,
		})
		//c.String(200, fmt.Sprint(book))
		return
	}
	c.JSON(400, gin.H{
		"error msg": err.Error(),
	})

}

func returnBook(c *gin.Context) {
	var history *RentalHistory
	t := time.Now().Add(time.Minute * 360)
	err := c.Bind(&history)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error msg": err.Error()})
		return
	}
	history.ReturnDate = t
	_, err = db.QueryOne(history, `UPDATE rental_history SET return_date = (?return_date) WHERE book_id = (?book_id) AND rental_date = (select max(rental_date) from rental_history where book_id = (?book_id)) RETURNING *`, history)
	if err == nil {
		c.JSON(400, gin.H{
			"result": history,
		})
		return
	}
	c.JSON(400, gin.H{
		"error msg": err.Error(),
	})

}

func rentABook(readerId int64, bookId int64) error{

	_, err := db.Exec(`BEGIN`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`UPDATE book SET current_reader = (?) WHERE book_id = (?)`, readerId,bookId)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		INSERT INTO rental_history (reader_id,book_id) VALUES (?,?)`, readerId, bookId)
	if err != nil {
		return err
	}

	_, err = db.Exec(`COMMIT`)
	if err != nil {
		return err
	}
	fmt.Println(readerId, " c книгой ", bookId, " добавлен")
	return nil
}

func updateBook(c *gin.Context) {
	var book *Book
	err := c.Bind(&book)
	if err != nil {
		c.JSON(400, gin.H{
			"error msg": err.Error(),
		})
	}
	_, err = db.QueryOne(book, `UPDATE book SET name = (?name) WHERE book_id = (?book_id) RETURNING *`, book)
	if err == nil {
		c.String(200, fmt.Sprint(book.BookId, " ", book.Name, " изменен успешно"))
		return
	}
	c.JSON(400, gin.H{
		"error msg": err.Error(),
	})

}

func deleteBook(c *gin.Context) {

	var book *Book
	err := c.Bind(&book)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error msg": err.Error()})
		return
	}
	_, err = db.QueryOne(book, `DELETE FROM book WHERE book_id = ? AND current_reader IS NULL RETURNING *`, book.BookId)
	if err == nil {
		c.String(200, fmt.Sprint(book.BookId, " ", book.Name, " удален успешно"))
		return
	}
	if err.Error() == "pg: no rows in result set" {
		c.String(200, fmt.Sprint("Такой книги не существует/Нельзя удалить книгу с действующим читателем"))
		return
	}

	c.JSON(400, gin.H{
		"error msg": err.Error(),
	})

}

func showBooks(c *gin.Context) {

	if c.Keys["role"] != "librarian"{
		c.JSON(http.StatusBadRequest, gin.H{"Недостаточно прав": ""})
		return}

	var book []*bookSearch
	var params searchParams
	var err error
	queryOffset := c.Query("offset")
	params.Status = c.Query("status")
	params.OrderBy = c.Query("order")
	params.Author = c.Query("author")
	if queryOffset != ""{
		params.Offset, err = strconv.Atoi(queryOffset)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error msg": err.Error()})
			return}
	}
	mainQueryBody := "SELECT book_id,release_date,current_reader,name AS book, genre,author_name AS author, image_filepath FROM book INNER JOIN genre ON genre.genre_id = book.genre_id INNER JOIN author ON author.author_id = book.author_id"
	queryEnd := " LIMIT 20 OFFSET (?offset)"

	if params.Author != "" || params.Status != "" {
		mainQueryBody += " WHERE"
	}
	//if params.Author != ""{
	//	mainQueryBody += " author.author_name LIKE '%(?author.author_name)%'"   error msg": "ERROR #42601 syntax error at or near \"Cobb\""???
	//}
	if params.Status == "rented" {
		mainQueryBody += " current_reader IS NOT NULL"
	}
	if params.Status == "free" {
		mainQueryBody += " current_reader IS NULL"
	}
	if params.OrderBy == "" {
		mainQueryBody += " ORDER BY book_id"
	}
	if params.OrderBy == "genre" {
		mainQueryBody += " ORDER BY genre"
	}
	if params.OrderBy == "author" {
		mainQueryBody += " ORDER BY author"
	}
	mainQueryBody += queryEnd
	fmt.Println(mainQueryBody)
	fmt.Println(params.Author)
	_, err = db.Query(&book, mainQueryBody, params)
	if err == nil {
		c.JSON(http.StatusOK, gin.H{"result": book, "params": params})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{
		"error msg": err.Error(),
	})
	return
	}

func createBook(c *gin.Context) {
	var bookAndFiles struct {
		book Book
		bookFile  *multipart.FileHeader
		bookImage *multipart.FileHeader
	}

	err := c.Bind(&bookAndFiles.book)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	fmt.Println(&bookAndFiles)
	bookAndFiles.bookFile, err = c.FormFile("book")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})
		return
	}

	filePathForBook := "nginx-1.21.1/resources/books/" + bookAndFiles.bookFile.Filename
	bookAndFiles.bookImage, err = c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})
		return
	}

	filePathForImage := "nginx-1.21.1/resources/images/" + bookAndFiles.bookImage.Filename

	if strings.ToLower(filepath.Ext(bookAndFiles.bookFile.Filename)) != ".pdf"{
		c.JSON(http.StatusBadRequest, gin.H{
			"Неверный формат файла книги": "",
		})
		return
	}
	if strings.ToLower(filepath.Ext(bookAndFiles.bookImage.Filename)) !=".jpg"{
		c.JSON(http.StatusBadRequest, gin.H{
			"Неверный формат файла обложки книги": "",
		})
		return
	}
	if err := c.SaveUploadedFile(bookAndFiles.bookImage, filePathForImage); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message":err.Error(),
		})
		return
	}

	if err := c.SaveUploadedFile(bookAndFiles.bookFile, filePathForBook); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message":err.Error(),
		})
		return
	}
	bookAndFiles.book.BookFilepath = filePathForBook
	bookAndFiles.book.ImageFilepath = filePathForImage
	_, err = db.QueryOne(&bookAndFiles.book, `
		INSERT INTO book (name,author_id,genre_id,release_date,book_filepath,image_filepath) VALUES (?name,?author_id,?genre_id,?release_date,?book_filepath,?image_filepath)`, bookAndFiles.book)
	if err == nil {
		c.JSON(http.StatusOK, gin.H{
			bookAndFiles.book.Name: "Книга добавлена успешно",
			filePathForImage:filePathForBook,
		})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{
		"error msg": err.Error(),
	})
}

func updateReader(c *gin.Context) {
	var reader *Reader
	err := c.Bind(&reader)
	if err != nil {
		c.JSON(400, gin.H{
			"error msg": err.Error(),
		})
	}
	_, err = db.QueryOne(reader, `UPDATE reader SET name = (?name) WHERE reader_id = (?reader_id) RETURNING *`, reader)
	if err == nil {
		c.String(200, fmt.Sprint(reader.ReaderId, " ", reader.Name, " изменен успешно"))
		return
	}
	c.JSON(400, gin.H{
		"error msg": err.Error(),
	})

}

func deleteReader(c *gin.Context) {

	var reader *Reader
	err := c.Bind(&reader)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error msg": err.Error()})
		return
	}
	_, err = db.QueryOne(reader, `DELETE FROM reader r WHERE r.reader_id = ? AND NOT EXISTS 
(SELECT 1 FROM book b WHERE r.reader_id = b.current_reader AND r.reader_id = ?) RETURNING *`, reader.ReaderId, reader.ReaderId)
	if err == nil {
		c.String(200, fmt.Sprint(reader.ReaderId, " ", reader.Name, " удален успешно"))
		return
	}
	c.JSON(400, gin.H{
		"error msg": err.Error(),
	})

}

func allReaders(c *gin.Context) {

	var reader []Reader
	err := c.Bind(&reader)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error msg": err.Error()})
		return
	}
	_, err = db.Query(&reader, `SELECT * FROM reader`)
	if err == nil {
		c.JSON(200, gin.H{
			"result": reader,
		})
		return
	}
	c.JSON(400, gin.H{
		"error msg": err.Error(),
	})
}

func createReader(c *gin.Context) {
	var reader *Reader
	err := c.Bind(&reader)
	if err != nil {
		panic(err)
	}
	_, err = db.QueryOne(reader, `
		INSERT INTO reader (name,birth_date) VALUES (?name,?birth_date) RETURNING reader_id`, reader)
	if err == nil {
		c.String(200, fmt.Sprint(reader.ReaderId, " ", reader.Name, " добавлен успешно"))
		return
	}
	c.JSON(400, gin.H{
		"error msg": err.Error(),
	})

}

func updateGenre(c *gin.Context) {
	var genre *Genre
	err := c.Bind(&genre)
	if err != nil {
		c.JSON(400, gin.H{
			"error msg": err.Error(),
		})
	}
	_, err = db.QueryOne(genre, `UPDATE genre SET genre = (?genre) WHERE genre_id = (?genre_id) RETURNING *`, genre)
	if err == nil {
		c.String(200, fmt.Sprint(genre.GenreId, " ", genre.Genre, " изменен успешно"))
		return
	}
	c.JSON(400, gin.H{
		"error msg": err.Error(),
	})
}

func deleteGenre(c *gin.Context) {
	var genre *Genre
	err := c.Bind(&genre)
	if err != nil {
		c.JSON(400, gin.H{
			"error msg": err.Error(),
		})
	}
	_, err = db.QueryOne(genre, `DELETE FROM genre g WHERE g.genre_id = ? AND NOT EXISTS 
(SELECT 1 FROM book b WHERE g.genre_id = b.genre_id AND g.genre_id = ?) RETURNING *`, genre.GenreId, genre.GenreId)
	if err.Error() == "pg: no rows in result set" {
		c.String(200, fmt.Sprint("Такого жанра не существует/Нельзя удалить жанр с существующими книгами"))
		return
	}
	if err == nil {
		c.String(200, fmt.Sprint(genre.GenreId, " ", genre.Genre, " удален успешно"))
		return
	}
	c.JSON(400, gin.H{
		"error msg": err.Error(),
	})
}

func createGenre(c *gin.Context) {
	var genre *Genre
	err := c.Bind(&genre)
	if err != nil {
		panic(err)
	}
	_, err = db.QueryOne(genre, `
		INSERT INTO genre (genre) VALUES (?) RETURNING genre_id`, genre.Genre)
	if err == nil {
		c.String(200, fmt.Sprint(genre.GenreId, " ", genre.Genre, " добавлен успешно"))
		return
	}
	c.JSON(400, gin.H{
		"error msg": err.Error(),
	})

}

func allGenres(c *gin.Context) {
	var genre []Genre
	err := c.Bind(&genre)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error msg": err.Error()})
		return
	}
	string2 := "SELECT * "
	string2 += "FROM genre"
	_, err = db.Query(&genre, string2)
	if err == nil {
		c.JSON(400, genre)
		return
	}
	c.JSON(400, gin.H{
		"error msg": err.Error(),
	})
}

func updateAuthor(c *gin.Context) {
	var authorID *Author
	err := c.Bind(&authorID)
	if err != nil {
		c.JSON(400, gin.H{
			"error msg": err.Error(),
		})
	}
	_, err = db.QueryOne(authorID, `UPDATE author SET author_name = (?author_name) WHERE author_id = (?author_id) RETURNING *`, authorID)
	if err == nil {
		c.String(200, fmt.Sprint(authorID.AuthorId, " ", authorID.AuthorName, " изменен успешно"))
		return
	}
	c.JSON(400, gin.H{
		"error msg": err.Error(),
	})

}
func deleteAuthor(c *gin.Context) {
	var authorID *Author
	err := c.Bind(&authorID)
	if err != nil {
		c.JSON(400, gin.H{
			"error msg": err.Error(),
		})
	}
	_, err = db.QueryOne(authorID, `DELETE FROM author a WHERE a.author_id = ? AND NOT EXISTS 
(SELECT 1 FROM book b WHERE a.author_id = b.author_id AND a.author_id = ?) RETURNING *`, authorID.AuthorId, authorID.AuthorId)
	if err == nil {
		c.String(200, fmt.Sprint(authorID.AuthorId, " ", authorID.AuthorName, " удален успешно"))
		return
	}
	c.JSON(400, gin.H{
		"error msg": err.Error(),
	})
}

func createAuthor(c *gin.Context) {

	var authorName *Author
	err := c.Bind(&authorName)
	if err != nil {
		panic(err)
	}
	_, err = db.QueryOne(authorName, `
		INSERT INTO author (author_name) VALUES (?author_name) RETURNING author_id`, authorName)
	if err == nil {
		c.String(200, fmt.Sprint(authorName.AuthorId, " ", authorName.AuthorName, " добавлен успешно"))
		return
	}
	c.JSON(400, gin.H{
		"error msg": err.Error(),
	})
}

func allAuthors(c *gin.Context) {

	authors, err := GetUsers(db)
	if err != nil {
		panic(err)
	}
	c.String(200, fmt.Sprint(authors))

}

func GetUsers(db *pg.DB) ([]string, error) {
	var authors []string
	_, err := db.Query(&authors, `SELECT * FROM author`)
	return authors, err
}

func GetUsersByIds(db *pg.DB, ids []int64) ([]string, error) {
	var authors []string
	_, err := db.Query(&authors, `SELECT * FROM author WHERE author_id IN (?)`, pg.In(ids))
	return authors, err
}

func CreateUser(db *pg.DB, author *Author) error {
	_, err := db.QueryOne(author, `
		INSERT INTO author (author_name) VALUES (?author_name) RETURNING author_id`, author)
	return err
}