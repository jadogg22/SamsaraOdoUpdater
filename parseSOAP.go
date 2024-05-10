package main

import (
	"encoding/xml"
	"fmt"
)

type TranBlock struct {
	XMLName xml.Name `xml:"tranBlock"`
	Trans   []Tran   `xml:",any"`
}
type Tran struct {
	ID      string `xml:"ID,attr"`
	Content []byte `xml:",innerxml"`
}

type TranContent struct {
	EventTS   string    `xml:"eventTS"`
	Equipment Equipment `xml:"equipment"`
	Position  Position  `xml:"position"`
	DriverID  string    `xml:"driverID"`
	Odometer  float64   `xml:"odometer"`
}

type Equipment struct {
	ID          string `xml:"ID,attr"`
	EquipType   string `xml:"equipType,attr"`
	UnitAddress string `xml:"unitAddress,attr"`
	MobileType  string `xml:"mobileType,attr"`
}

type Position struct {
	Lon   string `xml:"lon,attr"`
	Lat   string `xml:"lat,attr"`
	PosTS string `xml:"posTS,attr"`
}

func parseSoap(xmlData string, transMap map[string]TranContent) map[string]TranContent {

	var tranBlock TranBlock
	err := xml.Unmarshal([]byte(xmlData), &tranBlock)
	if err != nil {
		fmt.Println("Error parsing XML:", err)
		return nil
	}

	for _, tran := range tranBlock.Trans {

		// Parse the inner XML to get the actual fields
		var tranContent TranContent
		err := xml.Unmarshal(tran.Content, &tranContent)
		if err != nil {
			fmt.Println("Error parsing tran content:", err)
			continue
		}

		existingTran, ok := transMap[tranContent.Equipment.ID]
		if ok {
			if tranContent.Odometer > existingTran.Odometer {
				transMap[tranContent.Equipment.ID] = tranContent
			}
			// if the odometer is lower then do nothing
		} else {
			// No existing TranContet found, add it to the map
			transMap[tranContent.Equipment.ID] = tranContent
		}

	}
	return transMap

}

var ownerOperated = []string{"CHESS", "DEVAN1", "DJOHNSO1", "IPARSONS", "ISHARP", "JHAMILT1", "JMILL1", "JRODRIG1", "PMAPA", "RCOCHRAN", "RHUMME1", "WGAEDK1"}

func isOwnerOperated(driverID string) bool {
	for _, id := range ownerOperated {
		if id == driverID {
			return true
		}
	}
	return false
}
