// Command dbclean removes a farmer row by mobile number, cascading to their
// farms, crops, irrigation, soil and consent records.
//
// Used to tidy up after manual smoke-testing against a real database.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/SanTiwari07/NDVI_satellite/internal/config"
	"github.com/SanTiwari07/NDVI_satellite/internal/db"
)

func main() {
	mobile := flag.String("mobile", "", "mobile_number of the farmer to delete")
	flag.Parse()
	if *mobile == "" {
		fmt.Fprintln(os.Stderr, "usage: dbclean -mobile=9000000077")
		os.Exit(2)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pool, err := db.New(ctx, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "db:", err)
		os.Exit(1)
	}
	defer pool.Close()

	tag, err := pool.Exec(ctx, `DELETE FROM farmers WHERE mobile_number = $1`, *mobile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "delete:", err)
		os.Exit(1)
	}
	fmt.Printf("deleted %d farmer row(s) for %s\n", tag.RowsAffected(), *mobile)
}
