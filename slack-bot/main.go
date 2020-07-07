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
	"gopkg.in/resty.v1"

	"github.com/thinhvoxuan/stravaapi"
)

type RefreshToken struct {
	TokenType    string `json:"token_type"`
	AccessToken  string `json:"access_token"`
	ExpiresAt    int    `json:"expires_at"`
	RefreshToken string `json:"refresh_token"`
}

func findMyClubs() (result string) {
	client, context := initClient()

	clubs, _, err := client.ClubsApi.GetLoggedInAthleteClubs(context, &stravaapi.GetLoggedInAthleteClubsOpts{
		Page:    optional.NewInt32(1),
		PerPage: optional.NewInt32(20),
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

func getPaceRunNumber(summaryActivity stravaapi.SummaryActivity) (pace string) {
	paceNumber := float32(summaryActivity.MovingTime) / (summaryActivity.Distance / 1000)
	pace = getTimeFormat(int32(paceNumber))
	return
}

func getPaceSwimNumber(summaryActivity stravaapi.SummaryActivity) (pace string) {
	paceNumber := float32(summaryActivity.MovingTime) / (summaryActivity.Distance / 100)
	pace = getTimeFormat(int32(paceNumber))
	return
}

func getVelocity(summaryActivity stravaapi.SummaryActivity) string {
	v := (summaryActivity.Distance / 1000) / (float32(summaryActivity.MovingTime) / 3600)
	return fmt.Sprintf("%.2f", v)
}

func getReport(summaryActivity stravaapi.SummaryActivity) (text string) {
	switch *summaryActivity.Type_ {
	case stravaapi.RIDE:
		username := getUserName(summaryActivity)
		text = fmt.Sprintf("%s - %s - %s\n", username, summaryActivity.Name, time.Now().Format("January 02 2006"))
		text += fmt.Sprintf("`%s duration`\n", getTimeFormat(summaryActivity.MovingTime))
		text += fmt.Sprintf("`%.2f km/%s km/h speed`\n", summaryActivity.Distance/1000, getVelocity(summaryActivity))
		text += fmt.Sprintf("`%.2f ft/%.2f m climbed`\n", summaryActivity.TotalElevationGain*3.2808399, summaryActivity.TotalElevationGain)
	case stravaapi.SWIM:
		username := getUserName(summaryActivity)
		text = fmt.Sprintf("%s - %s - %s\n", username, summaryActivity.Name, time.Now().Format("January 02 2006"))
		text += fmt.Sprintf("`%s duration`\n", getTimeFormat(summaryActivity.MovingTime))
		text += fmt.Sprintf("`%.0f m/%s pace`\n", summaryActivity.Distance, getPaceSwimNumber(summaryActivity))
	default:
		username := getUserName(summaryActivity)
		text = fmt.Sprintf("%s - %s - %s\n", username, summaryActivity.Name, time.Now().Format("January 02 2006"))
		text += fmt.Sprintf("`%s duration`\n", getTimeFormat(summaryActivity.MovingTime))
		text += fmt.Sprintf("`%.2f km/%s pace`\n", summaryActivity.Distance/1000, getPaceRunNumber(summaryActivity))
		text += fmt.Sprintf("`%.2f ft/%.2f m climbed`\n", summaryActivity.TotalElevationGain*3.2808399, summaryActivity.TotalElevationGain)
	}

	return
}

func pushToSlack(summaryActivity stravaapi.SummaryActivity) bool {
	webhookURL := os.Getenv("SLACK_HOOK_URL")
	text := getReport(summaryActivity)
	payload := slack.Payload{
		Text:      text,
		Username:  "Strava Bot",
		IconEmoji: ":runner:",
	}
	err := slack.Send(webhookURL, "", payload)
	if len(err) > 0 {
		fmt.Printf("error: %s\n", err)
		return false
	}
	return true
}

func requestToken() (token string) {
	params := map[string]string{
		"client_id":     os.Getenv("CLIENT_ID"),
		"client_secret": os.Getenv("CLIENT_SECRET"),
		"refresh_token": os.Getenv("REFRESH_TOKEN"),
		"grant_type":    "refresh_token",
	}
	refreshToken := RefreshToken{}
	_, err := resty.R().SetQueryParams(params).SetResult(&refreshToken).Post("https://www.strava.com/oauth/token")

	if err != nil {
		fmt.Println("request token fail")
		fmt.Println(err)
		return
	}
	fmt.Println("request token: " + refreshToken.AccessToken)
	token = refreshToken.AccessToken
	return
}

func initClient() (client *stravaapi.APIClient, auth context.Context) {
	token := requestToken()
	auth = context.WithValue(context.Background(), stravaapi.ContextAccessToken, token)
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
