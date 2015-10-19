package data

import (
	".././dao"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

func GetUserData(userID int, toGet string) (int, error) {
	db, err := sql.Open("sqlite3", dao.DatabaseURL)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	var value int
	db.QueryRow("SELECT ? FROM user_data WHERE user_id = ?", toGet, userID).Scan(&value)

	return value, err
}

func GetUserStatistics(userID int) (submitted, accepted, attempted int) {
	db, err := sql.Open("sqlite3", dao.DatabaseURL)
	if err != nil {
		return
	}
	defer db.Close()

	db.QueryRow("SELECT submitted_count, accepted_count, attempted_count FROM user_data WHERE user_id = ?", userID).Scan(&submitted, &accepted, &attempted)
	return
}
