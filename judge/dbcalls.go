package judge

import (
	".././dao"
	".././problems"
	".././skills"
	".././users"
	"errors"
	"fmt"
	"time"
)

func AddProblem(problem problems.Problem) (err error) {

	db, err := dao.Open()
	if err != nil {
		return
	}

	tx, err := db.Begin()
	if err != nil {
		return
	}

	result, err := tx.Exec("INSERT INTO problems (title, description, difficulty, skill_id, uva_id, time_limit, memory_limit, sample_input, sample_output) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		problem.Title, problem.Description, problem.Difficulty, problem.SkillID, problem.UvaID, problem.TimeLimit, problem.MemoryLimit, problem.SampleInput, problem.SampleOutput)
	if err != nil {
		tx.Rollback()
		return
	}

	problemID, err := result.LastInsertId()
	if err != nil {
		tx.Rollback()
		return
	}
	if problem.UvaID == "" {
		_, err = tx.Exec("INSERT INTO inputoutput (problem_id, input_number, input, output) VALUES (?, ?, ?, ?)",
			problemID, 1, problem.Input, problem.Output)

		if err != nil {
			tx.Rollback()
			return
		}
	}

	tags := "INSERT INTO tags (problem_id, tag) VALUES "
	for i, tag := range problem.Tags {
		if i == len(problem.Tags)-1 {
			tags += " (" + fmt.Sprint(problemID) + `, "` + tag + `"); `
		} else {
			tags += " (" + fmt.Sprint(problemID) + `, "` + tag + `"), `
		}
	}
	if problem.Tags != nil {
		_, err = tx.Exec(tags)
		if err != nil {
			tx.Rollback()
			return
		}
	}
	tx.Commit()
	return
}

func editProblem(problem problems.Problem) error {

	db, err := dao.Open()
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec("UPDATE problems SET title = ?, description = ?, difficulty = ?, skill_id = ?, uva_id = ?, time_limit = ?, memory_limit = ?, sample_input = ?, sample_output = ? WHERE id = ?",
		problem.Title, problem.Description, problem.Difficulty, problem.SkillID, problem.UvaID, problem.TimeLimit, problem.MemoryLimit, problem.SampleInput, problem.SampleOutput, problem.Index)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec("UPDATE inputoutput SET input = ?, output = ? WHERE problem_id = ? AND input_number = ?",
		problem.Input, problem.Output, problem.Index, 1)

	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec("DELETE FROM tags WHERE problem_id = ?", problem.Index)

	if err != nil {
		tx.Rollback()
		return err
	}

	tags := "INSERT INTO tags (problem_id, tag) VALUES "
	for i, tag := range problem.Tags {
		if i == len(problem.Tags)-1 {
			tags += ` (` + fmt.Sprint(problem.Index) + `, "` + tag + `"); `
		} else {
			tags += ` (` + fmt.Sprint(problem.Index) + `, "` + tag + `"), `
		}
	}
	if problem.Tags != nil {
		_, err = tx.Exec(tags)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	tx.Commit()
	return nil
}

func deleteProblem(problemID int) (err error) {

	db, err := dao.Open()
	if err != nil {
		return
	}

	tx, err := db.Begin()
	if err != nil {
		return
	}

	_, err = tx.Exec("DELETE FROM problems WHERE id = ?", problemID)
	if err != nil {
		tx.Rollback()
		return
	}

	_, err = tx.Exec("DELETE FROM inputoutput where problem_id = ? AND input_number = ?",
		problemID, 1)

	if err != nil {
		tx.Rollback()
		return
	}

	return tx.Commit()
}

func addSubmission(submission Submission, userID int) (int, error) {
	db, err := dao.Open()
	if err != nil {
		return -1, err
	}

	if _, err := GetProblem(submission.ProblemIndex); err != nil {
		return -1, errors.New("No such problem")
	}
	result, err := db.Exec("INSERT INTO submissions (problem_id, user_id, directory, verdict, timestamp, language) VALUES (?, ?, ?, ?, ?, ?)",
		submission.ProblemIndex, userID, submission.Directory, submission.Verdict, time.Now(), submission.Language)

	if err != nil {
		return -1, err
	}

	submissionID, err := result.LastInsertId()

	if err != nil {
		return -1, err
	}

	return int(submissionID), nil
}

func getSubmissions(limit, offset int) (submissions []Submission, count int, err error) {
	db, err := dao.Open()
	if err != nil {
		return
	}

	rows, err := db.Query(`SELECT submissions.id, problem_id, title, username, verdict, user_account.id, IFNULL(runtime, 0), IFNULL(uva_submission_id, 0), language 
                        FROM problems, submissions, user_account
                        WHERE submissions.problem_id = problems.id AND user_account.id = submissions.user_id
                        ORDER BY timestamp DESC
                        LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return
	}
	for rows.Next() {
		var submission Submission
		err = rows.Scan(&submission.ID, &submission.ProblemIndex, &submission.ProblemTitle, &submission.Username, &submission.Verdict, &submission.UserID, &submission.Runtime, &submission.UvaSubmissionID, &submission.Language)
		if err != nil {
			return
		}
		submissions = append(submissions, submission)
	}

	err = db.QueryRow(`SELECT COUNT(*) 
                        FROM problems, submissions, user_account
                        WHERE submissions.problem_id = problems.id AND user_account.id = submissions.user_id`).Scan(&count)
	if err != nil {
		return
	}
	return
}

func GetSubmission(id int) (submission Submission, err error) {
	db, err := dao.Open()
	if err != nil {
		return
	}
	err = db.QueryRow(`SELECT submissions.id, problem_id, username, verdict, user_account.id, uva_submission_id, runtime, language 
              FROM submissions, user_account 
              WHERE user_account.id = submissions.user_id and submissions.id = ?`, id).Scan(&submission.ID, &submission.ProblemIndex,
		&submission.Username, &submission.Verdict, &submission.UserID, &submission.UvaSubmissionID, &submission.Runtime, &submission.Language)

	return
}

func usedSubmissionID(id int) (bool, error) {
	db, err := dao.Open()
	if err != nil {
		return true, err
	}
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM submissions WHERE uva_submission_id = ?", id).Scan(&count)
	if err != nil {
		return false, err
	}
	if count == 0 {
		return false, err
	} else {
		return true, err
	}
}

func acceptedAlready(userID, problemID int) (bool, error) {
	db, err := dao.Open()
	if err != nil {
		return false, err
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM submissions, user_account "+
		"WHERE user_account.id = submissions.user_id AND verdict = ?"+
		"AND submissions.problem_id = ? AND user_id = ?", problems.Accepted, problemID, userID).Scan(&count)

	if err != nil || count == 0 {
		return false, err
	}

	return true, err
}

func UpdateVerdictInDB(id int, verdict string) error {
	db, err := dao.Open()
	if err != nil {
		return err
	}

	_, err = db.Exec("UPDATE submissions SET verdict = ? WHERE id = ?", verdict, id)

	if err != nil {
		return err
	}

	return nil
}

func UpdateRuntime(id int, runtime float64) error {
	db, err := dao.Open()
	if err != nil {
		return err
	}

	_, err = db.Exec("UPDATE submissions SET runtime = ? WHERE id = ?", runtime, id)

	if err != nil {
		return err
	}

	return nil
}

func updateUvaSubmissionID(id, submissionID int) error {
	db, err := dao.Open()
	if err != nil {
		return err
	}

	_, err = db.Exec("UPDATE submissions SET uva_submission_id = ? WHERE id = ?", submissionID, id)

	if err != nil {
		return err
	}

	return nil
}

func GetProblems() (problemList []problems.Problem, err error) {
	db, err := dao.Open()
	if err != nil {
		return
	}
	rows, err := db.Query("SELECT id, title, description, difficulty, skill_id, time_limit, memory_limit, sample_input, sample_output, uva_id FROM problems")

	if err != nil {
		return
	}

	for rows.Next() {
		var problem problems.Problem
		err = rows.Scan(&problem.Index, &problem.Title, &problem.Description, &problem.Difficulty, &problem.SkillID, &problem.TimeLimit,
			&problem.MemoryLimit, &problem.SampleInput, &problem.SampleOutput, &problem.UvaID)
		problemList = append(problemList, problem)
		if err != nil {
			return
		}
	}

	return
}

func GetRelatedProblems(userID, problemID int) (relatedProblems []problems.Problem, err error) {
	db, err := dao.Open()
	if err != nil {
		return
	}
	rows, err := db.Query(`SELECT id, title, description, difficulty, skill_id, time_limit, memory_limit, sample_input, sample_output, uva_id FROM problems WHERE id IN (
                          SELECT DISTINCT problem_id FROM tags WHERE problem_id != ? AND tag IN 
                          (SELECT DISTINCT tag FROM problems, tags WHERE problem_id = id AND id = ?)
                          AND problem_id NOT IN (
                          SELECT DISTINCT problem_id 
                          FROM submissions 
                          WHERE verdict = ? AND user_id = ?)
                        );`, problemID, problemID, problems.Accepted, userID)
	if err != nil {
		return
	}

	unlocked, err := skills.GetUnlockedSkills(userID)
	if err != nil {
		return
	}
	for rows.Next() {
		var problem problems.Problem
		err = rows.Scan(&problem.Index, &problem.Title, &problem.Description, &problem.Difficulty, &problem.SkillID, &problem.TimeLimit,
			&problem.MemoryLimit, &problem.SampleInput, &problem.SampleOutput, &problem.UvaID)
		if unlocked[problem.SkillID] {
			relatedProblems = append(relatedProblems, problem)
		}
	}

	return
}

func GetProblem(index int) (problems.Problem, error) {
	db, err := dao.Open()
	var problem problems.Problem
	if err != nil {
		return problem, err
	}
	err = db.QueryRow(`SELECT id, title, description, difficulty, skill_id, time_limit, memory_limit, sample_input, sample_output, IFNULL(input, ""), IFNULL(output, ""), uva_id 
                     FROM problems LEFT JOIN inputoutput ON (problems.id = inputoutput.problem_id)
                     WHERE problems.id = ?`, index).Scan(&problem.Index, &problem.Title, &problem.Description,
		&problem.Difficulty, &problem.SkillID, &problem.TimeLimit, &problem.MemoryLimit, &problem.SampleInput,
		&problem.SampleOutput, &problem.Input, &problem.Output, &problem.UvaID)

	if err != nil {
		return problem, errors.New("No such problem")
	}

	rows, err := db.Query("SELECT tag FROM tags WHERE problem_id = ?", index)

	if err != nil {
		return problem, err
	}
	var tags []string
	var tag string
	for rows.Next() {
		rows.Scan(&tag)
		tags = append(tags, tag)
	}
	if len(tags) > 0 {
		problem.Tags = tags
	} else {
		problem.Tags = nil
	}
	return problem, nil
}

func GetUnsolvedTriedProblems(userID int) (unsolvedProblems []int, err error) {
	db, err := dao.Open()
	if err != nil {
		return
	}

	rows, err := db.Query(`SELECT DISTINCT problem_id FROM submissions WHERE user_id = ? AND 
                        problem_id NOT IN (SELECT DISTINCT problem_id FROM submissions WHERE user_id = ? AND verdict = ?);`,
		userID, userID, problems.Accepted)
	for rows.Next() {
		var problem int
		err = rows.Scan(&problem)
		if err != nil {
			return
		}
		unsolvedProblems = append(unsolvedProblems, problem)
	}
	return
}

func GetUnsolvedProblems(userID int) (unsolvedProblems []int, err error) {
	db, err := dao.Open()
	if err != nil {
		return
	}

	rows, err := db.Query(`SELECT DISTINCT id FROM problems 
                        WHERE id NOT IN(SELECT DISTINCT problem_id AS id 
                        FROM submissions 
                        WHERE user_id = ? AND verdict = ?);`,
		userID, problems.Accepted)
	if err != nil {
		return
	}
	for rows.Next() {
		var problem int
		err = rows.Scan(&problem)
		if err != nil {
			return
		}
		unsolvedProblems = append(unsolvedProblems, problem)
	}
	return
}

func GetUserWhoRecentlySolvedProblem(userID, problemID int) (user users.UserData, err error) {
	db, err := dao.Open()
	if err != nil {
		return
	}

	err = db.QueryRow(`SELECT user_account.id, username, email FROM user_account, submissions 
                      WHERE user_account.id = submissions.user_id AND submissions.problem_id = ? AND verdict = ? AND submissions.user_id != ?
                      ORDER BY timestamp DESC;`, problemID, problems.Accepted, userID).Scan(&user.ID, &user.Username, &user.Email)

	return
}

func getSubmissionsReceivedAndInqueue() (submissions []Submission, err error) {
	db, err := dao.Open()
	if err != nil {
		return
	}

	rows, err := db.Query(`SELECT submissions.id, problem_id, title, username, verdict, user_account.id, IFNULL(runtime, 0), 
                          IFNULL(uva_submission_id, 0), directory, language   
                         FROM problems, submissions, user_account 
                         WHERE submissions.problem_id = problems.id AND user_account.id = submissions.user_id AND verdict IN (?, ?, ?, ?)
                         ORDER BY timestamp DESC`, problems.Received, problems.Inqueue, problems.Compiling, problems.Running)

	if err != nil {
		return
	}
	for rows.Next() {
		var submission Submission
		err = rows.Scan(&submission.ID, &submission.ProblemIndex, &submission.ProblemTitle, &submission.Username, &submission.Verdict, &submission.UserID,
			&submission.Runtime, &submission.UvaSubmissionID, &submission.Directory, &submission.Language)
		if err != nil {
			return
		}
		submissions = append(submissions, submission)
	}
	return
}
