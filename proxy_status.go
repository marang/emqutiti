package emqutiti

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/marang/emqutiti/proxy"
	"github.com/marang/emqutiti/ui"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// startProxyStatusLogger logs proxy status immediately and at each interval.
func startProxyStatusLogger(addr string) func() {
	logProxyStatus(addr)
	ticker := time.NewTicker(time.Minute)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				logProxyStatus(addr)
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()
	return func() { close(done) }
}

func logProxyStatus(addr string) {
	if addr == "" {
		msg := fmt.Sprintf("%s proxy addr not configured", time.Now().Format(time.RFC3339))
		log.Println(lipgloss.NewStyle().Foreground(ui.ColWarn).Render(msg))
		return
	}
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		msg := fmt.Sprintf("%s proxy unreachable: %v", time.Now().Format(time.RFC3339), err)
		log.Println(lipgloss.NewStyle().Foreground(ui.ColRed).Render(msg))
		return
	}
	defer conn.Close()
	client := proxy.NewDBProxyClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	st, err := client.Status(ctx, &proxy.StatusRequest{})
	if err != nil {
		msg := fmt.Sprintf("%s proxy status error: %v", time.Now().Format(time.RFC3339), err)
		log.Println(lipgloss.NewStyle().Foreground(ui.ColWarn).Render(msg))
		return
	}
	var infos []string
	for _, db := range st.GetDbs() {
		infos = append(infos, fmt.Sprintf("%s/%s=%dB/%d", db.GetProfile(), db.GetBucket(), db.GetSize(), db.GetEntries()))
	}
	msg := fmt.Sprintf("%s clients:%d published:%d subscribed:%d deletes:%d %s", time.Now().Format(time.RFC3339), st.GetClients(), st.GetWrites(), st.GetReads(), st.GetDeletes(), strings.Join(infos, " "))
	log.Println(lipgloss.NewStyle().Foreground(ui.ColCyan).Render(msg))
}
