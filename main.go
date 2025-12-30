package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/getlantern/systray"
	"github.com/go-toast/toast"
)

const (
	// FAB marketplace free assets URL
	fabFreeURL = "https://www.fab.com/search?price=free&sort=relevancy"
	// Unreal Marketplace free assets URL (legacy)
	unrealFreeURL = "https://www.unrealengine.com/marketplace/en-US/assets?tag=4910"
	// Check interval
	checkInterval = 1 * time.Hour
	// Data file for tracking seen assets
	dataFileName = "seen_assets.json"
)

type Asset struct {
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	Price     string    `json:"price"`
	FirstSeen time.Time `json:"first_seen"`
}

type AppData struct {
	SeenAssets map[string]Asset `json:"seen_assets"`
	LastCheck  time.Time        `json:"last_check"`
}

var (
	appData     AppData
	dataFile    string
	httpClient  *http.Client
	checkNowCh  = make(chan struct{}, 1)
	stopCh      = make(chan struct{})
)

func main() {
	// Setup data file path in user's app data
	appDataDir, err := os.UserConfigDir()
	if err != nil {
		appDataDir = "."
	}
	dataDir := filepath.Join(appDataDir, "UnrealFreeAssets")
	os.MkdirAll(dataDir, 0755)
	dataFile = filepath.Join(dataDir, dataFileName)

	// Initialize HTTP client with reasonable timeout
	httpClient = &http.Client{
		Timeout: 30 * time.Second,
	}

	// Load existing data
	loadData()

	// Start the systray
	systray.Run(onReady, onExit)
}

func onReady() {
	// Set up the systray icon and menu
	systray.SetIcon(getIcon())
	systray.SetTitle("Unreal Free Assets")
	systray.SetTooltip("Monitors FAB/Unreal Marketplace for free assets")

	mCheckNow := systray.AddMenuItem("Check Now", "Check for free assets immediately")
	mLastCheck := systray.AddMenuItem("Last check: Never", "Shows when last check occurred")
	mLastCheck.Disable()
	systray.AddSeparator()
	mOpenFab := systray.AddMenuItem("Open FAB Free Assets", "Open FAB marketplace in browser")
	mOpenUnreal := systray.AddMenuItem("Open Unreal Free Assets", "Open Unreal marketplace in browser")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit the application")

	// Start the background checker
	go backgroundChecker(mLastCheck)

	// Handle menu clicks
	go func() {
		for {
			select {
			case <-mCheckNow.ClickedCh:
				select {
				case checkNowCh <- struct{}{}:
				default:
				}
			case <-mOpenFab.ClickedCh:
				openBrowser("https://www.fab.com/search?price=free")
			case <-mOpenUnreal.ClickedCh:
				openBrowser("https://www.unrealengine.com/marketplace/en-US/assets?tag=4910")
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()

	// Do an initial check on startup
	go func() {
		time.Sleep(5 * time.Second) // Wait a bit for startup
		select {
		case checkNowCh <- struct{}{}:
		default:
		}
	}()
}

func onExit() {
	close(stopCh)
	saveData()
}

func backgroundChecker(lastCheckItem *systray.MenuItem) {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			checkForFreeAssets(lastCheckItem)
		case <-checkNowCh:
			checkForFreeAssets(lastCheckItem)
		case <-stopCh:
			return
		}
	}
}

func checkForFreeAssets(lastCheckItem *systray.MenuItem) {
	log.Println("Checking for free assets...")
	
	newAssets := []Asset{}

	// Check FAB marketplace
	fabAssets, err := scrapeFabFreeAssets()
	if err != nil {
		log.Printf("Error scraping FAB: %v", err)
	} else {
		for _, asset := range fabAssets {
			if _, seen := appData.SeenAssets[asset.URL]; !seen {
				asset.FirstSeen = time.Now()
				appData.SeenAssets[asset.URL] = asset
				newAssets = append(newAssets, asset)
			}
		}
	}

	// Check Unreal Marketplace (legacy)
	unrealAssets, err := scrapeUnrealFreeAssets()
	if err != nil {
		log.Printf("Error scraping Unreal Marketplace: %v", err)
	} else {
		for _, asset := range unrealAssets {
			if _, seen := appData.SeenAssets[asset.URL]; !seen {
				asset.FirstSeen = time.Now()
				appData.SeenAssets[asset.URL] = asset
				newAssets = append(newAssets, asset)
			}
		}
	}

	appData.LastCheck = time.Now()
	lastCheckItem.SetTitle(fmt.Sprintf("Last check: %s", appData.LastCheck.Format("15:04 Jan 2")))
	
	saveData()

	// Notify about new free assets
	if len(newAssets) > 0 {
		notifyNewAssets(newAssets)
	}

	log.Printf("Check complete. Found %d new free assets.", len(newAssets))
}

func scrapeFabFreeAssets() ([]Asset, error) {
	req, err := http.NewRequest("GET", fabFreeURL, nil)
	if err != nil {
		return nil, err
	}

	// Set headers to appear as a browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("FAB returned status code: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var assets []Asset

	// FAB uses dynamic content, so we look for common patterns
	// This selector may need adjustment as FAB's HTML structure evolves
	doc.Find("a[href*='/listings/']").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}
		
		title := strings.TrimSpace(s.Text())
		if title == "" {
			title = s.Find("img").AttrOr("alt", "Unknown Asset")
		}
		
		if title != "" && href != "" {
			fullURL := href
			if !strings.HasPrefix(href, "http") {
				fullURL = "https://www.fab.com" + href
			}
			assets = append(assets, Asset{
				Title: title,
				URL:   fullURL,
				Price: "Free",
			})
		}
	})

	return assets, nil
}

func scrapeUnrealFreeAssets() ([]Asset, error) {
	req, err := http.NewRequest("GET", unrealFreeURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Unreal Marketplace returned status code: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var assets []Asset

	// Look for asset cards in Unreal Marketplace
	doc.Find("article.asset-card, div[data-component='asset-card'], a[href*='/product/']").Each(func(i int, s *goquery.Selection) {
		var title, href string
		
		// Try to find the link
		link := s.Find("a").First()
		if link.Length() == 0 {
			href, _ = s.Attr("href")
		} else {
			href, _ = link.Attr("href")
		}
		
		// Try to find the title
		titleEl := s.Find("h3, h2, .asset-title, [class*='title']").First()
		if titleEl.Length() > 0 {
			title = strings.TrimSpace(titleEl.Text())
		}
		if title == "" {
			title = strings.TrimSpace(s.Find("img").AttrOr("alt", ""))
		}

		if title != "" && href != "" {
			fullURL := href
			if !strings.HasPrefix(href, "http") {
				fullURL = "https://www.unrealengine.com" + href
			}
			assets = append(assets, Asset{
				Title: title,
				URL:   fullURL,
				Price: "Free",
			})
		}
	})

	return assets, nil
}

func notifyNewAssets(assets []Asset) {
	var message string
	if len(assets) == 1 {
		message = fmt.Sprintf("New free asset: %s", assets[0].Title)
	} else if len(assets) <= 3 {
		titles := make([]string, len(assets))
		for i, a := range assets {
			titles[i] = a.Title
		}
		message = fmt.Sprintf("New free assets:\n%s", strings.Join(titles, "\n"))
	} else {
		message = fmt.Sprintf("%d new free assets available!", len(assets))
	}

	notification := toast.Notification{
		AppID:   "Unreal Free Assets Monitor",
		Title:   "New Free Unreal Assets!",
		Message: message,
		Actions: []toast.Action{
			{Type: "protocol", Label: "Open FAB", Arguments: "https://www.fab.com/search?price=free"},
		},
	}

	if err := notification.Push(); err != nil {
		log.Printf("Failed to show notification: %v", err)
	}
}

func loadData() {
	appData.SeenAssets = make(map[string]Asset)

	data, err := os.ReadFile(dataFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Error reading data file: %v", err)
		}
		return
	}

	if err := json.Unmarshal(data, &appData); err != nil {
		log.Printf("Error parsing data file: %v", err)
	}
}

func saveData() {
	data, err := json.MarshalIndent(appData, "", "  ")
	if err != nil {
		log.Printf("Error serializing data: %v", err)
		return
	}

	if err := os.WriteFile(dataFile, data, 0644); err != nil {
		log.Printf("Error writing data file: %v", err)
	}
}

func openBrowser(url string) {
	cmd := exec.Command("cmd", "/c", "start", url)
	if err := cmd.Start(); err != nil {
		log.Printf("Error opening browser: %v", err)
	}
}

// getIcon returns the systray icon
func getIcon() []byte {
	return iconData
}
