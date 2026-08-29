package push

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sunvc/NoLets/common"
	"github.com/sunvc/apns2"
	"github.com/sunvc/apns2/payload"
	"github.com/sunvc/apns2/token"
)

func LocationPush(params *common.ParamsResult) map[string]string {
	pl := payload.NewPayload()
	// Add custom parameters
	for pair := params.Params.Oldest(); pair != nil; pair = pair.Next() {
		pl.Custom(pair.Key, pair.Value)
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs = make(map[string]string)
	for _, user := range params.Users {
		fmt.Println("user:", user)
		wg.Go(func() {
			if len(user.Location) == 0 {
				mu.Lock()
				errs[user.Key] = "no token specified"
				mu.Unlock()
				return
			}
			CLI := <-CLIENTS // Get a client from the pool
			CLIENTS <- CLI   // Put the client back into the pool

			// Create and send notification
			resp, err := CLI.Push(&apns2.Notification{
				DeviceToken: user.Location,
				CollapseID:  params.GetString(common.ID),
				Topic:       common.LocalConfig.Apple.Topic + ".location-query",
				Payload:     pl,
				Expiration:  common.DateNow().Add(10 * time.Minute),
				PushType:    apns2.PushTypeLocation,
			})

			// Error handling
			if err != nil {
				mu.Lock()
				errs[user.Key] = err.Error()
				mu.Unlock()
			} else if resp.StatusCode != 200 {
				mu.Lock()
				errs[user.Key] = fmt.Sprintf("APNs push failed: %s", resp.Reason)
				mu.Unlock()
			}
		})
	}

	wg.Wait()

	return errs
}

// Push message to APNs server
func Push(params *common.ParamsResult, pushType apns2.EPushType, token string) error {
	pl := payload.NewPayload().MutableContent()

	if pushType == apns2.PushTypeBackground {
		pl = pl.ContentAvailable()
	} else {

		pl = pl.AlertTitle(params.GetString(common.TITLE)).
			AlertSubtitle(params.GetString(common.SUBTITLE)).
			AlertBody(params.GetString(common.BODY)).
			Sound(params.GetString(common.SOUND)).
			TargetContentID(params.GetString(common.ID)).
			ThreadID(params.GetString(common.GROUP)).
			Category(params.GetString(common.CATEGORY))
	}

	// Add custom parameters
	for pair := params.Params.Oldest(); pair != nil; pair = pair.Next() {
		if _, skip := common.SkipKeys[pair.Key]; skip {
			continue
		}
		pl.Custom(pair.Key, pair.Value)
	}

	CLI := <-CLIENTS // Get a client from the pool
	CLIENTS <- CLI   // Put the client back into the pool

	// Create and send notification
	resp, err := CLI.Push(&apns2.Notification{
		DeviceToken: token,
		CollapseID:  params.GetString(common.ID),
		Topic:       common.LocalConfig.Apple.Topic,
		Payload:     pl,
		Expiration:  common.DateNow().Add(24 * time.Hour),
		PushType:    pushType,
	})

	// Error handling
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("APNs push failed: %s", resp.Reason)
	}
	return nil

}

func BatchPush(params *common.ParamsResult, pushType apns2.EPushType) gin.H {

	errors := make(gin.H, 0)
	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)

	for _, user := range params.Users {
		/// 1.25 版本新语法
		wg.Go(func() {
			if err := Push(params, pushType, user.Token); err != nil {
				log.Println(err.Error())
				mu.Lock()
				errors[user.Key] = err.Error()
				mu.Unlock()
			}
		})
	}

	wg.Wait()

	return errors
}

func GetToken() (auth string, expirted int64) {
	CLI := <-CLIENTS // Get a client from the pool
	CLIENTS <- CLI   // Put the client back into the pool
	return CLI.Token.GenerateIfExpired(), CLI.Token.IssuedAt + token.TokenTimeout
}

func PttPush(url string, token string) error {
	pl := payload.NewPayload().ContentAvailable()

	pl.Custom("url", url)

	CLI := <-CLIENTS // Get a client from the pool
	CLIENTS <- CLI   // Put the client back into the pool

	// Create and send notification
	resp, err := CLI.Push(&apns2.Notification{
		DeviceToken: token,
		Topic:       common.LocalConfig.Apple.Topic + ".voip-ptt",
		Payload:     pl,
		Expiration:  common.DateNow().Add(60 * time.Second),
		PushType:    apns2.PushTypePushToTalk,
		Priority:    apns2.PriorityHigh,
	})

	// Error handling
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("APNs push failed: %s", resp.Reason)
	}
	return nil
}
