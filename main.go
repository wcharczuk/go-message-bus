package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"os"

	"github.com/blendlabs/go-request"
	"github.com/wcharczuk/go-web"
)

var (
	_sharedSecret []byte
	_slackWebhook string
)

func slackWebhook() string {
	if len(_slackWebhook) == 0 {
		_slackWebhook = os.Getenv("SLACK_WEBHOOK")
	}
	return _slackWebhook
}

func sharedSecret() []byte {
	if len(_sharedSecret) == 0 {
		envSecret := os.Getenv("SHARED_SECRET")
		_sharedSecret, _ = base64.StdEncoding.DecodeString(envSecret)
	}

	return _sharedSecret
}

func verifyWebHook(action web.ControllerAction) web.ControllerAction {
	return func(rc *web.RequestContext) web.ControllerResult {
		if len(sharedSecret()) == 0 {
			return action(rc)
		}

		shopifyHeader := rc.Request.Header.Get("HTTP_X_SHOPIFY_HMAC_SHA256")
		if len(shopifyHeader) == 0 {
			return rc.API().BadRequest("missing `HTTP_X_SHOPIFY_HMAC_SHA256` header.")
		}

		compare, err := base64.StdEncoding.DecodeString(shopifyHeader)
		if err != nil {
			return rc.API().BadRequest(err.Error())
		}

		enc := hmac.New(sha256.New, sharedSecret())
		enc.Write(rc.PostBody())
		shouldBe := enc.Sum(nil)

		if !hmac.Equal(shouldBe, compare) {
			return rc.API().BadRequest("invalid `HTTP_X_SHOPIFY_HMAC_SHA256` header.")
		}

		return action(rc)
	}
}

func main() {
	app := web.New()
	app.SetName("Message Bus")
	//app.SetLogger(web.NewStandardOutputLogger())

	app.GET("/", func(rc *web.RequestContext) web.ControllerResult {
		return rc.JSON(map[string]string{"status": "ok!"})
	})

	app.POST("/shopper", func(rc *web.RequestContext) web.ControllerResult {
		var parsed map[string]interface{}
		err := rc.PostBodyAsJSON(&parsed)
		if err != nil {
			return rc.API().BadRequest(err.Error())
		}

		hookContents := map[string]interface{}{
			"text": fmt.Sprintf(
				`New Shopper Signup!
                <https://kissandwear.com/admin/customers/%v|%v> %v %v`,
				parsed["id"],
				parsed["email"],
				parsed["first_name"],
				parsed["last_name"],
			),
			"username": "Shopify (New Customer)",
			"icon_url": "https://support.wombat.co/hc/en-us/article_attachments/200579685/shopify-expert-web-designer.jpg",
		}

		err = request.NewHTTPRequest().AsPost().WithURL(slackWebhook()).WithJSONBody(hookContents).Execute()
		if err != nil {
			return rc.API().InternalError(err)
		}

		return rc.JSON(map[string]string{"status": "ok!"})
	}, verifyWebHook)

	app.POST("/order", func(rc *web.RequestContext) web.ControllerResult {
		var parsed map[string]interface{}
		err := rc.PostBodyAsJSON(&parsed)
		if err != nil {
			return rc.API().BadRequest(err.Error())
		}

		hookContents := map[string]interface{}{
			"text": fmt.Sprintf(
				`:moneybag: New Sale!
                <https://kissandwear.com/admin/orders/%v|%v> for <http://kissandwear.com/admin/customers/%v|%v>`,
				parsed["id"],
				parsed["total_price"],
				readMap(parsed, "customer", "id"),
				readMap(parsed, "customer", "email"),
			),
			"username": "Shopify (New Customer)",
			"icon_url": "https://support.wombat.co/hc/en-us/article_attachments/200579685/shopify-expert-web-designer.jpg",
		}

		err = request.NewHTTPRequest().AsPost().WithURL(slackWebhook()).WithJSONBody(hookContents).Execute()
		if err != nil {
			return rc.API().InternalError(err)
		}

		return rc.JSON(map[string]string{"status": "ok!"})
	}, verifyWebHook)

	log.Fatal(app.Start())
}

func readMap(contents map[string]interface{}, keys ...string) interface{} {
	var workingContents = contents
	var result interface{}

	for _, key := range keys {
		if tempResult, hasResult := workingContents[key]; hasResult {
			result = tempResult
			if typed, isTyped := result.(map[string]interface{}); isTyped {
				workingContents = typed
			} else {
				break
			}
		} else {
			break
		}
	}

	return result
}
