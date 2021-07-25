package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-pg/pg"
	"net/http"
	"strconv"
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
}

type searchParams struct {
	OrderBy string
	Offset  int `pg:"offset"`
	Status  string
	Author string `pg:"author.author_name"`
}

var db *pg.DB

func main() {

	db = pg.Connect(&pg.Options{
		User:     "postgres",
		Password: "admin",
		Database: "library",
	})
	defer db.Close()

	r := gin.Default()

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

	r.POST("/api/rentbook", rentABook)
	r.POST("/api/returnbook", returnBook)
	r.POST("/api/rentalhistory", showHistory)

	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")

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

func rentABook(c *gin.Context) {
	var book *Book
	var history []*RentalHistory

	err := c.Bind(&book)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error msg": err.Error()})
		return
	}

	_, err = db.Exec(`BEGIN`)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error msg": err.Error()})
		return
	}

	_, err = db.QueryOne(book, `UPDATE book SET current_reader = (?current_reader) WHERE book_id = (?book_id)`, book)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error msg": err.Error()})
		return
	}

	_, err = db.QueryOne(&history, `
		INSERT INTO rental_history (reader_id,book_id) VALUES (?,?) RETURNING rental_id`, book.CurrentReader, book.BookId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error msg": err.Error()})
		return
	}

	_, err = db.Exec(`COMMIT`)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"result": history})
	return
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
	mainQueryBody := "SELECT book_id,release_date,current_reader,name AS book, genre,author_name AS author FROM book INNER JOIN genre ON genre.genre_id = book.genre_id INNER JOIN author ON author.author_id = book.author_id"
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
	//	switch params.Status {
	//	case "Free":
	//		if params.OrderBy == "Genre" {
	//			_, err = db.Query(&book, `SELECT book_id,release_date,current_reader,name AS book, genre,author_name AS author FROM book
	//INNER JOIN genre ON genre.genre_id = book.genre_id INNER JOIN author ON author.author_id = book.author_id WHERE current_reader IS NULL ORDER BY genre LIMIT 20 OFFSET (?)`, params.Offset)
	//			if err == nil {
	//				c.JSON(200, gin.H{
	//					"result": book,
	//					"params": params,
	//				})
	//				return
	//			}
	//			c.JSON(400, gin.H{
	//				"error msg": err.Error(),
	//			})
	//			return
	//		}
	//		_, err = db.Query(&book, `SELECT book_id,release_date,current_reader,name AS book, genre,author_name AS author FROM book
	//INNER JOIN genre ON genre.genre_id = book.genre_id INNER JOIN author ON author.author_id = book.author_id WHERE current_reader IS NULL ORDER BY author LIMIT 20 OFFSET (?)`, params.Offset)
	//		if err == nil {
	//			c.JSON(200, gin.H{
	//				"result": book,
	//				"params": params,
	//			})
	//			return
	//		}
	//		c.JSON(400, gin.H{
	//			"error msg": err.Error(),
	//		})
	//	case "Rented":
	//		if params.OrderBy == "Genre" {
	//			_, err = db.Query(&book, `SELECT book_id,release_date,current_reader,name AS book, genre,author_name AS author FROM book
	//INNER JOIN genre ON genre.genre_id = book.genre_id INNER JOIN author ON author.author_id = book.author_id WHERE current_reader IS NOT NULL ORDER BY genre LIMIT 20 OFFSET (?)`, params.Offset)
	//			if err == nil {
	//				c.JSON(200, gin.H{
	//					"result": book,
	//					"params": params,
	//				})
	//				return
	//			}
	//			c.JSON(400, gin.H{
	//				"error msg": err.Error(),
	//			})
	//			return
	//		}
	//		_, err = db.Query(&book, `SELECT book_id,release_date,current_reader,name AS book, genre,author_name AS author FROM book
	//INNER JOIN genre ON genre.genre_id = book.genre_id INNER JOIN author ON author.author_id = book.author_id WHERE current_reader IS NOT NULL ORDER BY author LIMIT 20 OFFSET (?)`, params.Offset)
	//		if err == nil {
	//			c.JSON(200, gin.H{
	//				"result": book,
	//				"params": params,
	//			})
	//			return
	//		}
	//		c.JSON(400, gin.H{
	//			"error msg": err.Error(),
	//		})
	//	default:
	//		if params.OrderBy == "Genre" {
	//			_, err = db.Query(&book, `SELECT book_id,release_date,current_reader,name AS book, genre,author_name AS author FROM book
	//INNER JOIN genre ON genre.genre_id = book.genre_id INNER JOIN author ON author.author_id = book.author_id ORDER BY genre LIMIT 20 OFFSET (?)`, params.Offset)
	//			if err == nil {
	//				c.JSON(200, gin.H{
	//					"result": book,
	//					"params": params,
	//				})
	//				return
	//			}
	//			c.JSON(400, gin.H{
	//				"error msg": err.Error(),
	//			})
	//			return
	//		}
	//		_, err = db.Query(&book, `SELECT book_id,release_date,current_reader,name AS book, genre,author_name AS author FROM book
	//INNER JOIN genre ON genre.genre_id = book.genre_id INNER JOIN author ON author.author_id = book.author_id ORDER BY author LIMIT 20 OFFSET (?)`, params.Offset)
	//		if err == nil {
	//			c.JSON(200, gin.H{
	//				"result": book,
	//				"params": params,
	//			})
	//			return
	//		}
	//		c.JSON(400, gin.H{
	//			"error msg": err.Error(),
	//		})
	//	}


func createBook(c *gin.Context) {
	var book *Book
	err := c.Bind(&book)
	if err != nil {
		panic(err)
	}
	//if book.CurrentReader == nil{} если нет читателя
	_, err = db.QueryOne(book, `
		INSERT INTO book (name,author_id,genre_id,release_date) VALUES (?name,?author_id,?genre_id,?release_date) RETURNING book_id`, book)
	if err == nil {
		c.String(200, fmt.Sprint(book.BookId, " ", book.Name, " добавлен успешно"))
		return
	}
	c.JSON(400, gin.H{
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
	//if err.Error() == "pg: no rows in result set"{
	//	c.String(200, fmt.Sprint("Такого жанра не существует/Нельзя удалить жанр с существующими книгами"))
	//	return}
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
		c.JSON(400, gin.H{
			"result": genre,
		})
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

//
//func deleteAuthor(c *gin.Context) {
//	var authorID *Author
//	err := c.Bind(&authorID)
//	if err != nil {
//		c.JSON(400, gin.H{
//			"error msg": err.Error(),
//		})
//	}
//	_, err = db.QueryOne(authorID, `DELETE FROM author WHERE author_id = ? RETURNING *` , authorID.AuthorId)
//	if err == nil{
//		c.String(200, fmt.Sprint(authorID.AuthorId," ",authorID.AuthorName, " удален успешно"))
//		return}
//	c.JSON(400, gin.H{
//		"error msg": err.Error(),
//	})
//}

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
