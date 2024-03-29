package judge

import (
	".././achievements"
	".././helper"
	".././notifications"
	".././problems"
	".././skills"
	".././users"
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var DIR string
var OS string

type Judge interface {
	judge(s Submission)
}

type UvaJudge struct {
}

type CodeRangerJudge struct {
}

var uvaJudge = new(UvaJudge)
var codeRangerJudge = new(CodeRangerJudge)

type Submission struct {
	Username        string
	UserID          int
	ID              int
	ProblemIndex    int
	Directory       string `json:"-"`
	Verdict         string
	UvaSubmissionID int `json:"-"`
	Runtime         float64
	ProblemTitle    string
	Language        string
}

type VerdictData struct {
	Accepted          int
	WrongAnswer       int
	CompileError      int
	RuntimeError      int
	TimeLimitExceeded int
}

type UvaSubmissions struct {
	Name  string  `json:"name"`
	Uname string  `json:"uname"`
	Subs  [][]int `json:"subs"`
}

type UserSubmissions struct {
	Submissions UvaSubmissions `json:"821610"`
}

var UvaNodeDirectory string

const (
	Java = "Java"
	C    = "C"
)

const (
	UvaUsername = "CodeRanger2"
	UvaUserID   = "821610"
)

const (
	HardXP   = 50
	MediumXP = 30
	EasyXP   = 10
	Hard     = "Hard"
	Medium   = "Medium"
	Easy     = "Easy"
)

type Error struct {
	Verdict string
	Details string
}

func (e Error) Error() string {
	return e.Verdict // + ":\n" + e.Details
}

var (
	submissionQueue chan *Submission
	uvaQueue        chan *Submission
	cmd             *exec.Cmd
	stdin           io.WriteCloser
	stdout          bytes.Buffer
)

func InitQueues() {
	OS = runtime.GOOS
	if OS == "windows" {
		UvaNodeDirectory = `C:\uva-node`
	} else {
		UvaNodeDirectory = `/root/uva-node`
	}
	submissionQueue = make(chan *Submission)
	go func() {
		for s := range submissionQueue {
			p, err := GetProblem(s.ProblemIndex)
			if err != nil {
				log.Println(err)
			}

			if p.UvaID == "" {
				go codeRangerJudge.judge(s)
			} else {
				uvaJudge.judge(s)
			}
		}
	}()

	uvaQueue = make(chan *Submission)
	go func() {
		for s := range uvaQueue {
			go uvaJudge.checkVerdict(s)
		}
	}()
	cmd := exec.Command("npm", "start")
	cmd.Dir = UvaNodeDirectory
	cmd.Stdout = &stdout
	stdin, _ = cmd.StdinPipe()
	cmd.Start()
	io.WriteString(stdin, "add uva "+UvaUsername+" "+UvaUsername+"\n")
	if strings.Contains(stdout.String(), "is not recognized as an internal or external command,") ||
		strings.Contains(stdout.String(), "command not found") {
		log.Fatal("UVA NODE NOT FOUND!")
	}
	for {
		if strings.Contains(stdout.String(), "ERR!") {
			log.Fatal("UVA NODE NOT FOUND OR NPM INSTALL NOT YET RUN.")
		}
		if strings.Contains(stdout.String(), "Account added successfully") || strings.Contains(stdout.String(), "An existing account was replaced") {
			break
		}
	}
}

func (UvaJudge) checkVerdict(s *Submission) {
	// fmt.Println("checking")
	prob, err := GetProblem(s.ProblemIndex)
	// fmt.Println("http://uhunt.felix-halim.net/api/subs-nums/" + UvaUserID + "/" + prob.UvaID + "/" + strconv.Itoa(s.UvaSubmissionID - 1))
	resp, err := http.Get("http://uhunt.felix-halim.net/api/subs-nums/" + UvaUserID + "/" + prob.UvaID + "/" + strconv.Itoa(s.UvaSubmissionID-1))
	defer resp.Body.Close()
	if err != nil {
		uvaQueue <- s
	} else {
		userSubmissions := new(UserSubmissions)
		json.NewDecoder(resp.Body).Decode(userSubmissions)
		submissions := userSubmissions.Submissions
		for i := 0; i < len(submissions.Subs); i++ {
			if submissions.Subs[i][0] == s.UvaSubmissionID {
				if submissions.Subs[i][2] == 10 {
					submissionQueue <- s
				} else if submissions.Subs[i][2] == 20 || submissions.Subs[i][2] == 0 {
					time.Sleep(2 * time.Second)
					uvaQueue <- s
				} else {
					var verdict string
					switch submissions.Subs[i][2] {
					case 30:
						verdict = problems.CompileError
					case 35:
						verdict = problems.RestrictedFunction
					case 40:
						verdict = problems.RuntimeError
					case 45:
						verdict = problems.OutputLimitExceeded
					case 50:
						verdict = problems.TimeLimitExceeded
					case 60:
						verdict = problems.MemoryLimitExceeded
					case 70:
						verdict = problems.WrongAnswer
					case 80:
						verdict = problems.PresentationError
					case 90:
						verdict = problems.Accepted
					}
					s.Verdict = verdict
					s.Runtime = float64(submissions.Subs[i][3]) / 1000.00
					UpdateVerdict(s, verdict)
					err = UpdateRuntime(s.ID, s.Runtime)
					if err != nil {
						log.Println(err)
					}
					sendNotification(*s, prob)
				}
				break
			}
		}
	}
}

func sendNotification(s Submission, prob problems.Problem) {
	var relatedProblems []problems.Problem
	var newAchievements []achievements.Achievement
	var err error
	if s.Verdict == problems.Accepted {
		newAchievements, err = achievements.CheckNewAchievementsInSkill(s.UserID, s.ID, prob.SkillID)
		if err != nil {
			log.Println(err)
		}
	} else {
		relatedProblems, err = GetRelatedProblems(s.UserID, s.ProblemIndex)
		if err != nil {
			log.Println(err)
		}
	}
	user, err := users.GetUserData(s.UserID)
	if err != nil {
		log.Println(err)
	}
	skill, err := skills.GetUserDataOnSkill(s.UserID, prob.SkillID)
	if err != nil {
		log.Println(err)
	}
	problemList, err := skills.GetProblemsInSkill(prob.SkillID)
	if err != nil {
		log.Println(err)
	}
	skill.NumberOfProblems = len(problemList)
	data := struct {
		Submission      Submission
		Problem         problems.Problem
		User            users.UserData
		Skill           skills.Skill
		RelatedProblems []problems.Problem
		NewAchievements []achievements.Achievement
	}{
		s,
		prob,
		user,
		skill,
		relatedProblems,
		newAchievements,
	}
	message, err := json.Marshal(data)
	if err != nil {
		log.Println(err)
	} else {
		notifications.SendMessageTo(s.UserID, string(message), notifications.Notifications)
	}
}

func (UvaJudge) judge(s *Submission) {
	p, _ := GetProblem(s.ProblemIndex)

	io.WriteString(stdin, "use uva "+UvaUsername+"\n")
	var language string
	if s.Language == Java {
		language = "java"
	} else {
		language = "c"
	}

	str := "send " + p.UvaID + " " + filepath.Join(s.Directory, `Main.`+language) + "\n"

	io.WriteString(stdin, str)
	for !(strings.Contains(stdout.String(), "Send ok") || strings.Contains(stdout.String(), "send failed") ||
		strings.Contains(stdout.String(), "Login error")) {
		time.Sleep(2 * time.Second)
	}

	if strings.Contains(stdout.String(), "send failed") || strings.Contains(stdout.String(), "Login error") {
		stdout.Reset()
		go addToSubmissionQueue(s)
		return
	}

	stdout.Reset() // cleans out the stdout of the cmd to be used for another judging.

	time.Sleep(6 * time.Second)
	notgotten := true
	for notgotten {
		resp, err := http.Get("http://uhunt.felix-halim.net/api/subs-user-last/" + UvaUserID + "/1")

		if err == nil {
			defer resp.Body.Close()
			submissions := new(UvaSubmissions)
			err = json.NewDecoder(resp.Body).Decode(submissions)
			submissionID := submissions.Subs[0][0]
			used, err := usedSubmissionID(submissionID)
			if err != nil {
				log.Println(err)
				return
			}
			if used { // if the submission is used already that means uhunt is not updated yet. try again.
				continue
			}
			err = updateUvaSubmissionID(s.ID, submissionID)
			if err != nil {
				log.Println(err)
			}
			UpdateVerdict(s, problems.Inqueue)
			s.UvaSubmissionID = submissionID
			uvaQueue <- s
			notgotten = false
		}
	}

}

func addToSubmissionQueue(s *Submission) {
	submissionQueue <- s
}

func (CodeRangerJudge) judge(s *Submission) {
	var err *Error

	p, er := GetProblem(s.ProblemIndex)
	if er != nil {
		log.Println(er)
	}

	s.Verdict = problems.Compiling
	// UpdateVerdict(s, Compiling)

	err = s.compile()
	if err != nil {
		s.Verdict = err.Verdict
		UpdateVerdict(s, s.Verdict)
		sendNotification(*s, p)
		return
	}

	s.Verdict = problems.Running
	UpdateVerdict(s, problems.Running)
	t := time.Now()
	output, err := s.run(p)
	d := time.Now().Sub(t)
	UpdateRuntime(s.ID, helper.Truncate(d.Seconds(), 3))

	if err != nil {
		s.Verdict = err.Verdict
		UpdateVerdict(s, s.Verdict)
		sendNotification(*s, p)
		return
	}

	// s.Verdict = Judging
	// UpdateVerdict(s, Judging)

	if strings.Replace(output, "\r\n", "\n", -1) != strings.Replace(p.Output, "\r\n", "\n", -1) {
		// whitespace checks..? floats? etc.
		// fmt.Println(output)
		s.Verdict = problems.WrongAnswer
		UpdateVerdict(s, problems.WrongAnswer)
		sendNotification(*s, p)
		return
	}

	s.Verdict = problems.Accepted
	UpdateVerdict(s, problems.Accepted)
	sendNotification(*s, p)
}

func UpdateVerdict(s *Submission, verdict string) {
	err := UpdateVerdictInDB(s.ID, verdict)
	if err != nil {
		log.Println(err)
	}
	message, err := json.Marshal(s)
	if err != nil {
		log.Println(err)
	} else {
		notifications.SendMessageTo(s.UserID, string(message), notifications.Submissions)
	}
}

func (s Submission) compile() *Error {
	var stderr bytes.Buffer

	cmd := exec.Command("javac", "Main.java")
	cmd.Dir = s.Directory
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// fmt.Println(stderr.String())
		return &Error{problems.CompileError, stderr.String()}
	}

	return nil
}

func (s Submission) run(p problems.Problem) (string, *Error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := exec.Command("java", "-Djava.security.manager", "Main") // "-Xmx20m"
	cmd.Dir = s.Directory
	cmd.Stdin = strings.NewReader(p.Input)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	cmd.Start()
	timeout := time.After(time.Duration(p.TimeLimit) * time.Second)
	done := make(chan error)
	go func() { done <- cmd.Wait() }()
	select {
	case <-timeout:
		cmd.Process.Kill()
		return "", &Error{problems.TimeLimitExceeded, ""}
	case err := <-done:
		if err != nil {
			// fmt.Println(stderr.String())
			return "", &Error{problems.RuntimeError, stderr.String()}
		}
	}

	return stdout.String(), nil
}

func ResendReceivedAndCheckInqueue() (err error) {
	subs, err := getSubmissionsReceivedAndInqueue()
	if err != nil {
		return err
	}
	for _, sub := range subs {
		if sub.Verdict == problems.Inqueue {
			uvaQueue <- &sub
		} else if sub.Verdict == problems.Received ||
			sub.Verdict == problems.Compiling || sub.Verdict == problems.Running {
			submissionQueue <- &sub
		}
	}
	return
}
