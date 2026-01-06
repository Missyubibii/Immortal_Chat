package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
)

func main() {
	// 1. Kết nối MariaDB
	dsn := "root:root_password@tcp(chat_os_db:3306)/immortal_chat"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("Lỗi cấu hình DB:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Không thể kết nối MariaDB:", err)
	}
	fmt.Println("✅ 1. Kết nối MariaDB thành công!")

	// 2. Kết nối Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: "chat_os_redis:6379",
	})
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatal("Không thể kết nối Redis:", err)
	}
	fmt.Println("✅ 2. Kết nối Redis thành công!")

	// 3. Kích hoạt Watchdog
	startWatchdog(db)

	// 4. [QUAN TRỌNG] Khởi tạo Web Server để Tunnel kết nối
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello Immortal Chat! Tunnel is working.")
	})

	// Webhook Facebook yêu cầu phản hồi 200 OK
	http.HandleFunc("/webhook/facebook", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
		fmt.Println("📩 Đã nhận Webhook từ Facebook")
	})

	fmt.Println("🚀 Tế bào đang lắng nghe tại cổng :8080...")
	
	// Thay thế select{} bằng lệnh lắng nghe cổng thực tế
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("❌ Không thể khởi động Web Server:", err)
	}
}

// Watchdog logic: Tự động giải phóng bộ nhớ khi đầy (Chương 4.2)
func startWatchdog(db *sql.DB) {
	ticker := time.NewTicker(10 * time.Second) // Để 10s để bạn dễ quan sát khi test
	go func() {
		for range ticker.C {
			fmt.Println("🔍 Watchdog đang kiểm tra tài nguyên...")
			
			// Giả lập mức sử dụng ổ cứng vượt ngưỡng 70%
			diskUsage := 75 

			if diskUsage >= 70 {
				fmt.Println("⚠️ Disk > 70%, bắt đầu xả lũ (Purge)...")
				
				// Xóa cuốn chiếu dữ liệu cũ theo lô 1000 dòng
				result, err := db.Exec(`
					DELETE FROM webhook_logs 
					WHERE status = 'processed' 
					OR created_at < DATE_SUB(NOW(), INTERVAL 7 DAY) 
					LIMIT 1000`)
				
				if err == nil {
					rows, _ := result.RowsAffected()
					fmt.Printf("✅ Đã giải phóng %d bản ghi cũ.\n", rows)
				} else {
					fmt.Printf("❌ Lỗi khi thực hiện Purge: %v\n", err)
				}
			}
		}
	}()
}