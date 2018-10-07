package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/antihax/optional"
	"github.com/ashwanthkumar/slack-go-webhook"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/joho/godotenv"

	"github.com/thinhvoxuan/stravaapi"
)

func findMyClubs() (result string) {
	client, context := initClient()

	clubs, _, err := client.ClubsApi.GetLoggedInAthleteClubs(context, &stravaapi.GetLoggedInAthleteClubsOpts{
		Page:    optional.NewInt32(1),
		PerPage: optional.NewInt32(90),
	})

	if err != nil {
		fmt.Println(err)
		return
	}

	result = "Clubs: \n"
	for _, c := range clubs {
		result += " - " + c.Name + "\n"
	}

	return
}

func createKeySummaryActivities(summaryActivities stravaapi.SummaryActivity) (keyString string, keyBase64 string) {
	jsonEncode, err := json.Marshal(&summaryActivities)
	if err != nil {
		fmt.Println(err)
	}
	keyString = string(jsonEncode)
	hasher := sha1.New()
	hasher.Write(jsonEncode)
	keyBase64 = hex.EncodeToString(hasher.Sum(nil))
	return
}

func fetchClubsActivity(clubID int32, db *gorm.DB) (result string) {
	client, context := initClient()
	summaryActivities, _, err := client.ClubsApi.GetClubActivitiesById(context, clubID, &stravaapi.GetClubActivitiesByIdOpts{
		Page:    optional.NewInt32(1),
		PerPage: optional.NewInt32(20),
	})

	if err != nil {
		fmt.Println(err)
		return
	}

	result = "Activity: \n"
	for _, a := range summaryActivities {
		keyString, keyBase64 := createKeySummaryActivities(a)
		// Find key in DB if exits
		// If not end message into slack
		// Write record into DB

		var summaryActivityLog SummaryActivityLog
		var count int
		if db.Where("key_base64 = ?", keyBase64).First(&summaryActivityLog).Count(&count); count != 0 {
			fmt.Println("found record: ", count)
			continue
		}

		summaryActivityLog = SummaryActivityLog{
			KeyBase64: keyBase64,
			KeyString: keyString,
		}

		if err := db.Create(&summaryActivityLog).Error; err != nil {
			fmt.Println("cannot save record ", err)
			continue
		}
		if pushToSlack(a) == false {
			db.Delete(summaryActivityLog)
		}

		result += keyString + " " + keyBase64 + "\n"
	}
	return
}

func getUserName(summaryActivity stravaapi.SummaryActivity) string {
	username := summaryActivity.Athlete.Firstname + " " + summaryActivity.Athlete.Lastname
	return username
}

func getTimeFormat(time int32) (timeString string) {
	hrs := time / 3600
	minute := (time % 3600) / 60
	second := time % 60
	if hrs > 0 {
		timeString += fmt.Sprintf("%02d:%02d:%02d", hrs, minute, second)
	} else {
		timeString += fmt.Sprintf("%02d:%02d", minute, second)
	}
	return
}

func getPaceNumber(summaryActivity stravaapi.SummaryActivity) (pace string) {
	paceNumber := float32(summaryActivity.MovingTime) / (summaryActivity.Distance / 1000)
	pace = getTimeFormat(int32(paceNumber))
	fmt.Println(summaryActivity.Name, paceNumber, pace)
	return
}

func getReport(summaryActivity stravaapi.SummaryActivity) (title string, text string) {
	username := getUserName(summaryActivity)
	title = fmt.Sprintf("%s - %s in %s\n", username, summaryActivity.Name, time.Now().Format("January 02 2006"))
	text += fmt.Sprintf("*Distance:* %.2f KM\n", summaryActivity.Distance/1000)
	text += fmt.Sprintf("*Time:* %s\n", getTimeFormat(summaryActivity.MovingTime))
	text += fmt.Sprintf("*Pace:* %s\n", getPaceNumber(summaryActivity))
	text += fmt.Sprintf("*Climbed:* %.2fm\n", summaryActivity.TotalElevationGain)
	return
}

func pushToSlack(summaryActivity stravaapi.SummaryActivity) bool {
	webhookURL := os.Getenv("SLACK_HOOK_URL")
	title, text := getReport(summaryActivity)

	// text := "*Distance:* 11.22 KM\n" + "*Time:* 01:12:40\n" + "*Pace:* 6:29\n" + "*Climbed:* 90.8m\n"
	// title := "Hai Nhan L. - Morning Run"
	authorName := "Strava API"
	goodColor := "good"
	attachment1 := slack.Attachment{Color: &goodColor, Text: &text, Title: &title, AuthorName: &authorName}
	payload := slack.Payload{
		// Text:        report,
		Username:    "Strava Bot",
		IconEmoji:   ":runner:",
		Attachments: []slack.Attachment{attachment1},
	}
	err := slack.Send(webhookURL, "", payload)
	if len(err) > 0 {
		fmt.Printf("error: %s\n", err)
		return false
	}
	return true
}

func initClient() (client *stravaapi.APIClient, auth context.Context) {
	auth = context.WithValue(context.Background(), stravaapi.ContextAccessToken, os.Getenv("TOKEN"))
	cfg := stravaapi.NewConfiguration()
	client = stravaapi.NewAPIClient(cfg)
	return client, auth
}

func middlewareAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secret := r.URL.Query().Get("SECRET")
		if len(secret) == 0 || secret != os.Getenv("SECRET") {
			http.Error(w, http.StatusText(401), 401)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func initHTTP(db *gorm.DB) {
	myClubHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		result := findMyClubs()
		io.WriteString(w, result)
	})

	myActivityHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		clubIDString := req.URL.Query().Get("clubID")
		clubID, err := strconv.ParseInt(clubIDString, 10, 32)
		if err != nil {
			fmt.Println(err)
		}
		result := fetchClubsActivity(int32(clubID), db)
		io.WriteString(w, result)
	})

	http.Handle("/my-club", middlewareAuthentication(myClubHandler))
	http.Handle("/club-activity", middlewareAuthentication(myActivityHandler))
	fmt.Println(http.ListenAndServe(":8080", nil))
}

// SummaryActivityLog save into DB
type SummaryActivityLog struct {
	gorm.Model
	Name      string
	KeyBase64 string
	KeyString string
}

func initModel(db *gorm.DB) {
	db.AutoMigrate(&SummaryActivityLog{})
}

func main() {
	godotenv.Load()
	db, err := gorm.Open("postgres", "host=db user=postgres dbname=stravalog password=example sslmode=disable")
	defer db.Close()
	if err != nil {
		fmt.Println("Error-Connect-DB")
		fmt.Println(err)
	}
	initModel(db)
	initHTTP(db)
}
