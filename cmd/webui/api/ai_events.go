package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// aiEventsKeepalive re-sends a change event even when nothing was signalled, as
// a safety net: a subscriber registered between a mutation's database write and
// its notifyChange would otherwise wait for the next mutation to resync. Ten
// seconds is far above any real mutation gap during recognition, and the client
// skips re-rendering when the fetched data's stamp is unchanged, so an idle page
// costs one cheap pair of list reads per interval and zero DOM work.
const aiEventsKeepalive = 10 * time.Second

// HandleAIEvents is GET /api/ai/events: a Server-Sent Events stream that pushes
// one `change` event whenever task or suggestion state mutates (enqueue, claim,
// task end, suggestion edit/approve/reject). The event carries no payload the
// client acts on — it means "re-read the lists" — because the paged list
// endpoints remain the single source of truth for what the page shows.
//
// An initial event is sent on connect so a fresh or reconnected EventSource
// resyncs immediately without waiting for the next mutation.
func (rt *Router) HandleAIEvents(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	// Disables proxy buffering (nginx and friends) so events reach the browser
	// as they are written rather than in one buffered lump.
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)

	// Subscribe before the initial event so a mutation landing between the
	// first write and the registration cannot slip through unnoticed.
	changes, cancel := rt.AISvc.SubscribeChange()
	defer cancel()

	send := func() bool {
		if _, err := fmt.Fprintf(c.Writer, "event: change\ndata: %d\n\n", time.Now().UnixMilli()); err != nil {
			return false
		}
		c.Writer.Flush()
		return true
	}
	// retry tells the browser how long to wait before reconnecting a dropped
	// stream; the reconnect itself then resyncs via the initial event below.
	fmt.Fprint(c.Writer, "retry: 3000\n\n")
	if !send() {
		return
	}

	ticker := time.NewTicker(aiEventsKeepalive)
	defer ticker.Stop()

	ctx := c.Request.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-changes:
		case <-ticker.C:
		}
		if !send() {
			return
		}
	}
}