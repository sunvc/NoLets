package common

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sunvc/apns2"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func newTestParams() *ParamsResult {
	return &ParamsResult{Params: orderedmap.New[ParamName, interface{}]()}
}

func TestNormalizeKey(t *testing.T) {
	p := newTestParams()
	cases := map[string]ParamName{
		"Device-Token 2!": DEVICETOKEN + "2",
		"Body":            BODY,
		"MD":              MD,
		"a_b-c.d":         "abcd",
	}
	for in, want := range cases {
		if got := p.NormalizeKey(in); got != want {
			t.Errorf("NormalizeKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestConvenientParamsHandler(t *testing.T) {
	p := newTestParams()
	p.Params.Set(DATA, 123)
	p.Params.Set(MARKDOWN, "hello")
	p.Params.Set(SOUND, "bell")
	ConvenientParamsHandler(p.Params)

	if p.GetString(BODY) != "hello" {
		t.Fatalf("body = %q, want hello (markdown wins over data alias)", p.GetString(BODY))
	}
	if _, ok := p.Params.Get(DATA); ok {
		t.Error("DATA alias should be deleted after conversion")
	}
	if _, ok := p.Params.Get(MARKDOWN); ok {
		t.Error("MARKDOWN should be deleted after conversion")
	}
	if p.GetString(STYLE) != string(Markdown) {
		t.Errorf("style = %q, want markdown", p.GetString(STYLE))
	}
	if p.GetString(SOUND) != "bell.caf" {
		t.Errorf("sound = %q, want bell.caf", p.GetString(SOUND))
	}
}

func TestParamsNanAndDefault(t *testing.T) {
	// alert push: body present, defaults filled
	p := newTestParams()
	p.Params.Set(BODY, "hi")
	if pt := ParamsNanAndDefault(p); pt != apns2.PushTypeAlert {
		t.Errorf("pushType = %q, want alert", pt)
	}
	if p.GetString(CATEGORY) != string(MyNotificationCategory) {
		t.Errorf("category = %q, want default %q", p.GetString(CATEGORY), MyNotificationCategory)
	}
	if p.GetString(AUTOCOPY) != "0" || p.GetString(LEVEL) != "active" {
		t.Errorf("defaults not set: autocopy=%q level=%q", p.GetString(AUTOCOPY), p.GetString(LEVEL))
	}
	if p.GetString(ID) == "" {
		t.Error("id default not generated")
	}

	// background push: id only
	bg := newTestParams()
	bg.Params.Set(ID, "abc")
	if pt := ParamsNanAndDefault(bg); pt != apns2.PushTypeBackground {
		t.Errorf("pushType = %q, want background", pt)
	}

	// nothing to push -> sentinel "0"
	empty := newTestParams()
	if pt := ParamsNanAndDefault(empty); pt != apns2.EPushType("0") {
		t.Errorf("pushType = %q, want 0", pt)
	}
}

func TestHandlerParamsToMapOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// JSON body: device keys (string + array) must land in Keys, not just the map
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/push?devicekey=querykey",
		strings.NewReader(`{"devicekey":"bodykey1,bodykey2","devicekeys":["arrkey"],"title":"hi"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	p := newTestParams()
	p.HandlerParamsToMapOrder(c)

	if p.GetString(TITLE) != "hi" {
		t.Errorf("title = %q, want hi", p.GetString(TITLE))
	}
	for _, want := range []string{"querykey", "bodykey1", "bodykey2", "arrkey"} {
		if !contains(p.Keys, want) {
			t.Errorf("Keys = %v, missing %q", p.Keys, want)
		}
	}

	// Form body: regular fields must be plain strings (not []string -> "[v]")
	c2, _ := gin.CreateTestContext(httptest.NewRecorder())
	c2.Request = httptest.NewRequest("POST", "/push",
		strings.NewReader("title=hello&devicekey=formkey"))
	c2.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	p2 := newTestParams()
	p2.HandlerParamsToMapOrder(c2)

	if p2.GetString(TITLE) != "hello" {
		t.Errorf("form title = %q, want hello", p2.GetString(TITLE))
	}
	if !contains(p2.Keys, "formkey") {
		t.Errorf("Keys = %v, missing formkey", p2.Keys)
	}
}
