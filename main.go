package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	MockServerURL = "https://politrack.africa/api/votes"
	StatusURL     = "https://politrack.africa/api/votes/status"
	PollID        = 81
	CompetitorID  = 374
)

type VoterProfile struct {
	Name   string `json:"name"`
	Gender string `json:"gender"`
}

type Vote struct {
	ID           int    `json:"id"`
	CompetitorID int    `json:"competitorId"`
	VoterID      string `json:"voter_id"`
	Name         string `json:"name"`
	Gender       string `json:"gender"`
	Region       string `json:"region"`
	County       string `json:"county"`
	Constituency string `json:"constituency"`
	Ward         string `json:"ward"`
}

type LogEntry struct {
	Time    string `json:"time"`
	Message string `json:"message"`
	Type    string `json:"type"` // "info", "success", "skipped"
}

var (
	firstNames   []string
	lastNames    []string
	logs         []LogEntry
	totalSuccess int
	totalSkipped int
	mu           sync.Mutex
)

func main() {
	rand.Seed(time.Now().UnixNano())

	data, err := ioutil.ReadFile("first.json")
	if err != nil {
		log.Fatalf("Failed to read first.json: %v", err)
	}
	if err := json.Unmarshal(data, &firstNames); err != nil {
		log.Fatalf("Failed to parse first.json: %v", err)
	}

	data, err = ioutil.ReadFile("last.json")
	if err != nil {
		log.Fatalf("Failed to read last.json: %v", err)
	}
	if err := json.Unmarshal(data, &lastNames); err != nil {
		log.Fatalf("Failed to parse last.json: %v", err)
	}

	router := gin.Default()
	router.LoadHTMLGlob("templates/*")

	router.GET("/dashboard", func(c *gin.Context) {
		mu.Lock()
		defer mu.Unlock()
		c.HTML(http.StatusOK, "dashboard.html", gin.H{
			"logs":         logs,
			"totalVotes":   len(logs),
			"totalSuccess": totalSuccess,
			"totalSkipped": totalSkipped,
		})
	})

	go voteTicker()

	router.Run(":8000")
}

func voteTicker() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	endTime := time.Now().Add(1 * time.Hour)

	const workersPerTick = 50

	for now := range ticker.C {
		if now.After(endTime) {
			logMessage("Finished 18 hours of voting simulation.", "info")
			break
		}
		for i := 0; i < workersPerTick; i++ {
			go simulateVote()
		}
	}
}

func simulateVote() {
	voter := randomVoter()
	voterID := randomString(10)

	logMessage(fmt.Sprintf("Created voter: %s (%s) with ID %s", voter.Name, voter.Gender, voterID), "info")

	statusURL := fmt.Sprintf("%s?pollId=%d&voter_id=%s", StatusURL, PollID, voterID)
	resp, err := http.Get(statusURL)
	if err != nil {
		logMessage(fmt.Sprintf("Error checking vote status: %v", err), "info")
		return
	}
	defer resp.Body.Close()

	var statusResp struct {
		AlreadyVoted bool `json:"alreadyVoted"`
	}
	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, &statusResp)

	if statusResp.AlreadyVoted {
		logMessage(fmt.Sprintf("Voter %s already voted. Skipping.", voterID), "skipped")
		return
	}

	vote := Vote{
		ID:           PollID,
		CompetitorID: CompetitorID,
		VoterID:      voterID,
		Name:         voter.Name,
		Gender:       voter.Gender,
		Region:       "National",
		County:       "All",
		Constituency: "",
		Ward:         "",
	}

	payload, _ := json.Marshal(vote)
	req, _ := http.NewRequest("POST", MockServerURL, strings.NewReader(string(payload)))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp2, err := client.Do(req)
	if err != nil {
		logMessage(fmt.Sprintf("Error posting vote: %v", err), "info")
		return
	}
	defer resp2.Body.Close()

	body2, _ := ioutil.ReadAll(resp2.Body)
	var voteResp map[string]interface{}
	json.Unmarshal(body2, &voteResp)

	logMessage(fmt.Sprintf("Voted for %s. Response: %v", voter.Name, voteResp), "success")
}

func randomVoter() VoterProfile {
	first := firstNames[rand.Intn(len(firstNames))]
	last := lastNames[rand.Intn(len(lastNames))]
	gender := "Male"
	if rand.Intn(2) == 0 {
		gender = "Female"
	}
	return VoterProfile{
		Name:   fmt.Sprintf("%s %s", first, last),
		Gender: gender,
	}
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	sb := strings.Builder{}
	for i := 0; i < n; i++ {
		sb.WriteByte(letters[rand.Intn(len(letters))])
	}
	return sb.String()
}

func logMessage(msg string, typ string) {
	mu.Lock()
	defer mu.Unlock()
	entry := LogEntry{
		Time:    time.Now().Format("15:04:05"),
		Message: msg,
		Type:    typ,
	}
	logs = append([]LogEntry{entry}, logs...)

	if typ == "success" {
		totalSuccess++
	} else if typ == "skipped" {
		totalSkipped++
	}

	fmt.Println(entry.Time, "-", entry.Message)
}
