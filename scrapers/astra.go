/*
	This file contains the code for the Astra scraper.
*/

package scrapers

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/joho/godotenv"
)

func ScrapeAstra(outDir string) {

	// Load env vars
	if err := godotenv.Load(); err != nil {
		log.Panic("Error loading .env file")
	}

	// Start chromedp
	chromedpCtx, cancel := utils.InitChromeDp()
	defer cancel()
	fmt.Println("1")
	// Init http client
	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}
	fmt.Println("2")
	cli := &http.Client{Transport: tr}
	fmt.Println("3")

	/*astraHeaders := */
	utils.RefreshAstraToken(chromedpCtx)
	fmt.Println("4")
	url := fmt.Sprintf("https://www.aaiscloud.com/UTXDallas/~api/calendar/CalendarWeekGrid?_dc=%d&action=GET", time.Now().UnixMilli())
	body := "start=0&limit=5000&isForWeekView=false&fields=ActivityId%2CActivityPk%2CActivityName%2CParentActivityId%2CParentActivityName%2CMeetingType%2CDescription%2CStartDate%2CEndDate%2CDayOfWeek%2CStartMinute%2CEndMinute%2CActivityTypeCode%2CResourceId%2CCampusName%2CBuildingCode%2CRoomNumber%2CRoomName%2CLocationName%2CInstitutionId%2CSectionId%2CSectionPk%2CIsExam%2CIsCrosslist%2CIsAllDay%2CIsPrivate%2CEventId%2CEventPk%2CCurrentState%2CNotAllowedUsageMask%2CUsageColor%2CUsageColorIsPrimary%2CEventTypeColor%2CMaxAttendance%2CActualAttendance%2CCapacity&filter=(((StartDate%3C%3D%222024-09-26T23%3A00%3A00%22)%26%26(EndDate%3E%3D%222024-09-26T00%3A00%3A00%22))%26%26((((((((Resource.Building.CampusId%20in%20(%2203c9d930-7343-11e9-8a0c-35dcbeb1edcd%22))%26%26(Resource.Regions.Id%20in%20(%223578b3b0-9dab-11e9-bb13-b5bc7e192516%22)))%26%26(Resource.RoomTypeId%20in%20(%22fe74a890-65f8-11e9-991a-ff0e0065dfaa%22)))%26%26(((EventMeetingByActivityId.Event.EventTypeId%20in%20(%221a7720e9-8d19-11e9-b19f-0556148ced27%22%2C%221a7720ea-8d19-11e9-b19f-0556148ced27%22%2C%221a7720eb-8d19-11e9-b19f-0556148ced27%22%2C%221a7720ec-8d19-11e9-b19f-0556148ced27%22%2C%221a7720ed-8d19-11e9-b19f-0556148ced27%22%2C%221a7720ee-8d19-11e9-b19f-0556148ced27%22%2C%221a7720ef-8d19-11e9-b19f-0556148ced27%22%2C%221a7720f0-8d19-11e9-b19f-0556148ced27%22%2C%221a7720f1-8d19-11e9-b19f-0556148ced27%22%2C%221a7720f2-8d19-11e9-b19f-0556148ced27%22%2C%22874f9347-10f4-4367-ab1e-d697b187e9cb%22%2C%221a7720f4-8d19-11e9-b19f-0556148ced27%22%2C%221a7720f5-8d19-11e9-b19f-0556148ced27%22%2C%221a7720f6-8d19-11e9-b19f-0556148ced27%22%2C%221a7720e8-8d19-11e9-b19f-0556148ced27%22%2C%220494ce20-15e1-11ee-9d2b-ff74be387a2d%22%2C%221a7720f8-8d19-11e9-b19f-0556148ced27%22%2C%221a7720f9-8d19-11e9-b19f-0556148ced27%22))%26%26(CurrentState%20in%20(%22Incomplete%22%2C%22Requested%22%2C%22Scheduled%22)))%26%26(ActivityTypeCode%3D%3D2)))%7C%7C((((Resource.Building.CampusId%20in%20(%2203c9d930-7343-11e9-8a0c-35dcbeb1edcd%22))%26%26(Resource.Regions.Id%20in%20(%223578b3b0-9dab-11e9-bb13-b5bc7e192516%22)))%26%26(Resource.RoomTypeId%20in%20(%22fe74a890-65f8-11e9-991a-ff0e0065dfaa%22)))%26%26(ActivityTypeCode%3D%3D1)))%7C%7C(((((Resource.Building.CampusId%20in%20(%2203c9d930-7343-11e9-8a0c-35dcbeb1edcd%22))%26%26(Resource.Regions.Id%20in%20(%223578b3b0-9dab-11e9-bb13-b5bc7e192516%22)))%26%26(Resource.RoomTypeId%20in%20(%22fe74a890-65f8-11e9-991a-ff0e0065dfaa%22)))%26%26(((PrePostMeetingByActivityId.EventMeeting.Event.EventTypeId%20in%20(%221a7720e9-8d19-11e9-b19f-0556148ced27%22%2C%221a7720ea-8d19-11e9-b19f-0556148ced27%22%2C%221a7720eb-8d19-11e9-b19f-0556148ced27%22%2C%221a7720ec-8d19-11e9-b19f-0556148ced27%22%2C%221a7720ed-8d19-11e9-b19f-0556148ced27%22%2C%221a7720ee-8d19-11e9-b19f-0556148ced27%22%2C%221a7720ef-8d19-11e9-b19f-0556148ced27%22%2C%221a7720f0-8d19-11e9-b19f-0556148ced27%22%2C%221a7720f1-8d19-11e9-b19f-0556148ced27%22%2C%221a7720f2-8d19-11e9-b19f-0556148ced27%22%2C%22874f9347-10f4-4367-ab1e-d697b187e9cb%22%2C%221a7720f4-8d19-11e9-b19f-0556148ced27%22%2C%221a7720f5-8d19-11e9-b19f-0556148ced27%22%2C%221a7720f6-8d19-11e9-b19f-0556148ced27%22%2C%221a7720e8-8d19-11e9-b19f-0556148ced27%22%2C%220494ce20-15e1-11ee-9d2b-ff74be387a2d%22%2C%221a7720f8-8d19-11e9-b19f-0556148ced27%22%2C%221a7720f9-8d19-11e9-b19f-0556148ced27%22))%26%26(CurrentState%20in%20(%22Incomplete%22%2C%22Requested%22%2C%22Scheduled%22)))%26%26(ActivityTypeCode%3D%3D252)))%7C%7C((((Resource.Building.CampusId%20in%20(%2203c9d930-7343-11e9-8a0c-35dcbeb1edcd%22))%26%26(Resource.Regions.Id%20in%20(%223578b3b0-9dab-11e9-bb13-b5bc7e192516%22)))%26%26(Resource.RoomTypeId%20in%20(%22fe74a890-65f8-11e9-991a-ff0e0065dfaa%22)))%26%26(((SetupTeardownWindowByActivityId.EventMeeting.Event.EventTypeId%20in%20(%221a7720e9-8d19-11e9-b19f-0556148ced27%22%2C%221a7720ea-8d19-11e9-b19f-0556148ced27%22%2C%221a7720eb-8d19-11e9-b19f-0556148ced27%22%2C%221a7720ec-8d19-11e9-b19f-0556148ced27%22%2C%221a7720ed-8d19-11e9-b19f-0556148ced27%22%2C%221a7720ee-8d19-11e9-b19f-0556148ced27%22%2C%221a7720ef-8d19-11e9-b19f-0556148ced27%22%2C%221a7720f0-8d19-11e9-b19f-0556148ced27%22%2C%221a7720f1-8d19-11e9-b19f-0556148ced27%22%2C%221a7720f2-8d19-11e9-b19f-0556148ced27%22%2C%22874f9347-10f4-4367-ab1e-d697b187e9cb%22%2C%221a7720f4-8d19-11e9-b19f-0556148ced27%22%2C%221a7720f5-8d19-11e9-b19f-0556148ced27%22%2C%221a7720f6-8d19-11e9-b19f-0556148ced27%22%2C%221a7720e8-8d19-11e9-b19f-0556148ced27%22%2C%220494ce20-15e1-11ee-9d2b-ff74be387a2d%22%2C%221a7720f8-8d19-11e9-b19f-0556148ced27%22%2C%221a7720f9-8d19-11e9-b19f-0556148ced27%22))%26%26(CurrentState%20in%20(%22Incomplete%22%2C%22Requested%22%2C%22Scheduled%22)))%26%26(ActivityTypeCode%3D%3D251)))))%7C%7C(((((Resource.Building.CampusId%20in%20(%2203c9d930-7343-11e9-8a0c-35dcbeb1edcd%22))%26%26(Resource.Regions.Id%20in%20(%223578b3b0-9dab-11e9-bb13-b5bc7e192516%22)))%26%26(Resource.RoomTypeId%20in%20(%22fe74a890-65f8-11e9-991a-ff0e0065dfaa%22)))%26%26((ActivityTypeCode%3D%3D9)%26%26(ActivityId%3D%3Dnull)))%7C%7C((ActivityTypeCode%3D%3D356)%7C%7C(ActivityTypeCode%3D%3D357))))%7C%7C(ActivityTypeCode%3D%3D255)))&sortOrder=%2BStartDate%2C%2BStartMinute&page=1&group=%7B%22property%22%3A%22StartDate%22%2C%22direction%22%3A%22ASC%22%7D&sort=%5B%7B%22property%22%3A%22StartDate%22%2C%22direction%22%3A%22ASC%22%7D%2C%7B%22property%22%3A%22StartMinute%22%2C%22direction%22%3A%22ASC%22%7D%5D"
	req, err := http.NewRequest("POST", url, strings.NewReader(body))
	if err != nil {
		panic(err)
	}
	fmt.Println("5")
	//req.Header = astraHeaders
	res, err := cli.Do(req)
	if err != nil {
		panic(err)
	}
	fmt.Println("6")
	if res.StatusCode != 200 {
		log.Panicf("ERROR: Status was: %s\nIf the status is 404, you've likely been IP ratelimited!", res.Status)
	}
	fmt.Println("7")
}
