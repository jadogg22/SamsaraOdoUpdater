package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/joho/godotenv"
)

func main() {

	// Start Samsara Workflow
	fmt.Println("Grabbing Odometer Readings from Samsara")

	// get the spinner ready for nice looking UI
	done := make(chan struct{})
	go showSpinner(done)

	//Load .env file with samsaraKey
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Error loading .env file")
	}
	samsaraKey := os.Getenv("samsaraAccessToken")

	// Make request
	bodyBytes, err := SamsaraRequest(samsaraKey)
	if err != nil {
		fmt.Println("Unable to make request properly, exiting: ", err)
	}

	myVehicleData, err := ParseSamsaraResponse(bodyBytes)
	if err != nil {
		fmt.Println(err)
		return
	}

	myCount := len(myVehicleData)

	fmt.Println("total assets", myCount)
	// print a couple of them...
	if myCount > 3 {
		for i := 0; i <= 3; i++ {
			asset := myVehicleData[i]
			asset.Print()
		}
	}

	// stop the spinner
	done <- struct{}{}

	// --------------- start onmnitracs workflow ------------------
	// We're still not all the way done with omnitracks probably have another month worth
	// of data that we're collecting so I need to grab the data from this too.
	fmt.Println("Grabbing data from Omnitracks...")
	done = make(chan struct{}) // doing the same formating for this spinner
	go showSpinner(done)

	// grab the needed keys from the .env file
	username := os.Getenv("omniUsername")
	password := os.Getenv("omniPassword")

	lastTransaction := "0"
	transactions := "1"

	// create the hashmap of greatest odometer readings
	transMap := make(map[string]TranContent)
	for transactions != "" {
		transactions, newLastTransaction, err := createRequest(username, password, lastTransaction)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		// Update lastTransaction with the latest transaction ID
		lastTransaction = newLastTransaction

		if transactions == "" {
			fmt.Println("transactions is empty")
			break
		}

		transMap = parseSoap(transactions, transMap)
	}

	//I'm taking the code from the previous updater here but basically now I have two data structures, a map with the ID as the key and
	// then I just have an array of the data that likly doesnt have any duplactes so I think I'm going to do a reverse seach on the map by
	// looking if the samsara id is in the omni tracks data, if it is (either through it out, or compair the data and return the largest)
	// basically we're looking for the data in the transmap that doesnt exist in the samsara data. The data that in unique in OMNItracks
	// should append too the array to update.
	removeCount := 0
	//for loop over slice data and remove any dupe data
	for _, asset := range myVehicleData {
		if _, ok := transMap[asset.Name]; ok {
			removeCount++
			delete(transMap, asset.Name)
		}
	}
	fmt.Printf("Removed %d duplacates.\n", removeCount)

	for equipmentID, tranContent := range transMap {
		if isOwnerOperated(tranContent.DriverID) {
			//make sure we dont add any drivers we dont need
			continue
		}

		// errorhandling for something weird with lat and long
		lat, err := strconv.ParseFloat(tranContent.Position.Lat, 64)
		if err != nil {
			lat = 0.0
		}

		long, err := strconv.ParseFloat(tranContent.Position.Lon, 64)
		if err != nil {
			long = 0.0
		}

		asset := MyVehicleData{
			Name:      equipmentID,
			Timestamp: tranContent.EventTS,
			Odometer:  tranContent.Odometer,
			Place:     "nuh-uh", // no automatic places here
			Latitude:  lat,
			Longitude: long,
		}
		// add the asset to the array
		myVehicleData = append(myVehicleData, asset)
	}
	done <- struct{}{}

	// ---------- Dossier part -------------
	// We're not stayin with Dossier for vary much longer
	// so i'm not going to make it super beautifl or anything
	// although its basically going to be what it used to be.

	fmt.Println("Changing Odometers... (this may take a while)")
	numToChange := len(myVehicleData)

	numWorkers := 5
	accessToken, err := GetAccessToken()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	jobs := make(chan MyVehicleData, numToChange)
	for _, asset := range myVehicleData {
		jobs <- asset
	}
	close(jobs)
	var count int32
	for i := 0; i < numWorkers; i++ {
		go worker(&wg, jobs, accessToken, int(numToChange), &count)
	}

	wg.Wait()
	fmt.Println("Finished Updating Odometers!")
}

func showSpinner(done chan struct{}) {
	chars := []string{"|", "/", "-", "\\"}
	spinner := 0

	for {
		select {
		case <-done:
			fmt.Println("\rDone!")
			return
		default:
			fmt.Printf("\r%s", chars[spinner])
			spinner = (spinner + 1) % len(chars)
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func worker(wg *sync.WaitGroup, jobs <-chan MyVehicleData, accessToken string, total int, completed *int32) {
	defer wg.Done()
	for asset := range jobs {
		err := makeMeterChange(asset, accessToken)
		if err != nil {
			fmt.Printf("Error processing asset %s: %v\n", asset.Name, err)
		}
		atomic.AddInt32(completed, 1)
		fmt.Print(ProgressBar(total, completed))
	}
}

// ProgressBar generates a progress bar string based on the actual count and total count.
func ProgressBar(total int, actual *int32) string {
	const width = 50 // Width of the progress bar
	done := int(float64(*actual) / float64(total) * float64(width))

	bar := strings.Repeat("=", done) + ">" + strings.Repeat(" ", width-done)
	progress := fmt.Sprintf("[%s]", bar[:width])
	return fmt.Sprintf("\r%s (%d/%d) ", progress, *actual, total)
}
