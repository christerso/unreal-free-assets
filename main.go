package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/PuerkitoBio/goquery"
	"github.com/go-toast/toast"
)

const (
	fabFreeURL      = "https://www.fab.com/search?price=free&sort=relevancy"
	unrealFreeURL   = "https://www.unrealengine.com/marketplace/en-US/assets?tag=4910"
	unrealSourceURL = "https://unrealsource.com/dispatch/"
	checkInterval   = 1 * time.Hour
	dataFileName    = "seen_assets.json"
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
	appData      AppData
	dataFile     string
	dataDir      string
	httpClient   *http.Client
	fyneApp      fyne.App
	mainWindow   fyne.Window
	assetsList   *widget.List
	sortedAssets []Asset
	statusLabel  *widget.Label
)

// Custom dark theme with Unreal orange accent
type unrealTheme struct{}

func (t *unrealTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNamePrimary:
		return color.RGBA{245, 130, 32, 255}
	case theme.ColorNameBackground:
		return color.RGBA{26, 26, 46, 255}
	case theme.ColorNameButton:
		return color.RGBA{245, 130, 32, 255}
	case theme.ColorNameForeground:
		return color.RGBA{255, 255, 255, 255}
	case theme.ColorNameHover:
		return color.RGBA{60, 60, 80, 255}
	case theme.ColorNameSelection:
		return color.RGBA{245, 130, 32, 100}
	case theme.ColorNameInputBackground:
		return color.RGBA{40, 40, 60, 255}
	case theme.ColorNameSeparator:
		return color.RGBA{80, 80, 100, 255}
	default:
		return theme.DefaultTheme().Color(name, variant)
	}
}

func (t *unrealTheme) Font(style fyne.TextStyle) fyne.Resource   { return theme.DefaultTheme().Font(style) }
func (t *unrealTheme) Icon(name fyne.ThemeIconName) fyne.Resource { return theme.DefaultTheme().Icon(name) }
func (t *unrealTheme) Size(name fyne.ThemeSizeName) float32 {
	if name == theme.SizeNameText {
		return 14
	}
	return theme.DefaultTheme().Size(name)
}

func main() {
	// Setup paths
	appDataDir, _ := os.UserConfigDir()
	if appDataDir == "" {
		appDataDir = "."
	}
	dataDir = filepath.Join(appDataDir, "UnrealFreeAssets")
	os.MkdirAll(dataDir, 0755)
	dataFile = filepath.Join(dataDir, dataFileName)

	httpClient = &http.Client{Timeout: 30 * time.Second}
	loadData()
	initIcon()

	// Create Fyne app
	fyneApp = app.New()
	fyneApp.Settings().SetTheme(&unrealTheme{})

	// Setup system tray
	if desk, ok := fyneApp.(desktop.App); ok {
		setupSystemTray(desk)
	}

	// Create main window (hidden initially)
	mainWindow = fyneApp.NewWindow("Free Unreal Assets")
	mainWindow.Resize(fyne.NewSize(750, 550))
	mainWindow.SetCloseIntercept(func() {
		mainWindow.Hide()
	})

	buildMainUI()

	// Start background checker
	go backgroundChecker()

	// Initial check after delay
	go func() {
		time.Sleep(3 * time.Second)
		checkForFreeAssets()
	}()

	// Run (window stays hidden, systray active)
	fyneApp.Run()
}

func setupSystemTray(desk desktop.App) {
	// Create tray menu
	menu := fyne.NewMenu("Unreal Free Assets",
		fyne.NewMenuItem("View Found Assets", func() {
			refreshAssetsList()
			mainWindow.Show()
			mainWindow.RequestFocus()
		}),
		fyne.NewMenuItem("Check Now", func() {
			go checkForFreeAssets()
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Open FAB Marketplace", func() {
			openBrowser("https://www.fab.com/search?price=free")
		}),
		fyne.NewMenuItem("Open Unreal Source", func() {
			openBrowser("https://unrealsource.com/dispatch/")
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Quit", func() {
			saveData()
			fyneApp.Quit()
		}),
	)

	desk.SetSystemTrayMenu(menu)
	if iconData != nil && len(iconData) > 0 {
		desk.SetSystemTrayIcon(fyne.NewStaticResource("icon.png", iconData))
	}
}

func buildMainUI() {
	// Header
	title := canvas.NewText("Free Unreal Assets", color.RGBA{245, 130, 32, 255})
	title.TextSize = 26
	title.TextStyle = fyne.TextStyle{Bold: true}

	statusLabel = widget.NewLabel("Loading...")
	statusLabel.Alignment = fyne.TextAlignCenter

	header := container.NewVBox(
		container.NewCenter(title),
		statusLabel,
		widget.NewSeparator(),
	)

	// Assets list
	sortedAssets = getSortedAssets()

	assetsList = widget.NewList(
		func() int { return len(sortedAssets) },
		func() fyne.CanvasObject {
			titleLabel := widget.NewLabel("Asset Title Here")
			titleLabel.TextStyle = fyne.TextStyle{Bold: true}
			titleLabel.Wrapping = fyne.TextTruncate

			dateLabel := widget.NewLabel("Found: date")
			dateLabel.TextStyle = fyne.TextStyle{Italic: true}

			openBtn := widget.NewButton("Open →", func() {})
			openBtn.Importance = widget.HighImportance

			left := container.NewVBox(titleLabel, dateLabel)
			return container.NewBorder(nil, nil, nil, openBtn, left)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(sortedAssets) {
				return
			}
			asset := sortedAssets[id]

			c := obj.(*fyne.Container)
			left := c.Objects[0].(*fyne.Container)
			titleLabel := left.Objects[0].(*widget.Label)
			dateLabel := left.Objects[1].(*widget.Label)
			openBtn := c.Objects[1].(*widget.Button)

			displayTitle := asset.Title
			if len(displayTitle) > 70 {
				displayTitle = displayTitle[:67] + "..."
			}
			titleLabel.SetText(displayTitle)

			isNew := asset.FirstSeen.After(time.Now().Truncate(24 * time.Hour))
			dateText := fmt.Sprintf("Found: %s", asset.FirstSeen.Format("Jan 2, 2006 15:04"))
			if isNew {
				dateText = "🆕 " + dateText
			}
			dateLabel.SetText(dateText)

			url := asset.URL
			openBtn.OnTapped = func() { openBrowser(url) }
		},
	)

	assetsList.OnSelected = func(id widget.ListItemID) {
		if id < len(sortedAssets) {
			openBrowser(sortedAssets[id].URL)
		}
		assetsList.Unselect(id)
	}

	// Footer buttons
	checkBtn := widget.NewButton("🔄 Check Now", func() {
		go checkForFreeAssets()
	})

	fabBtn := widget.NewButton("🌐 Open FAB", func() {
		openBrowser("https://www.fab.com/search?price=free")
	})
	fabBtn.Importance = widget.HighImportance

	clearBtn := widget.NewButton("🗑 Clear History", func() {
		clearHistory()
		refreshAssetsList()
	})

	footer := container.NewVBox(
		widget.NewSeparator(),
		container.NewCenter(container.NewHBox(checkBtn, fabBtn, clearBtn)),
	)

	mainWindow.SetContent(container.NewBorder(header, footer, nil, nil, assetsList))
	updateStatusLabel()
}

func refreshAssetsList() {
	sortedAssets = getSortedAssets()
	if assetsList != nil {
		assetsList.Refresh()
	}
	updateStatusLabel()
}

func updateStatusLabel() {
	if statusLabel == nil {
		return
	}
	checkTime := "Never"
	if !appData.LastCheck.IsZero() {
		checkTime = appData.LastCheck.Format("Jan 2, 15:04")
	}
	statusLabel.SetText(fmt.Sprintf("%d assets found • Last check: %s", len(sortedAssets), checkTime))
}

func getSortedAssets() []Asset {
	assets := make([]Asset, 0, len(appData.SeenAssets))
	for _, asset := range appData.SeenAssets {
		assets = append(assets, asset)
	}
	sort.Slice(assets, func(i, j int) bool {
		return assets[i].FirstSeen.After(assets[j].FirstSeen)
	})
	return assets
}

func backgroundChecker() {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()
	for range ticker.C {
		checkForFreeAssets()
	}
}

func checkForFreeAssets() {
	log.Println("Checking for free assets...")
	newAssets := []Asset{}

	// Check FAB
	if assets, err := scrapeFabFreeAssets(); err != nil {
		log.Printf("FAB error: %v", err)
	} else {
		for _, a := range assets {
			if _, seen := appData.SeenAssets[a.URL]; !seen {
				a.FirstSeen = time.Now()
				appData.SeenAssets[a.URL] = a
				newAssets = append(newAssets, a)
			}
		}
	}

	// Check Unreal Marketplace
	if assets, err := scrapeUnrealFreeAssets(); err != nil {
		log.Printf("Unreal error: %v", err)
	} else {
		for _, a := range assets {
			if _, seen := appData.SeenAssets[a.URL]; !seen {
				a.FirstSeen = time.Now()
				appData.SeenAssets[a.URL] = a
				newAssets = append(newAssets, a)
			}
		}
	}

	// Check Unreal Source
	if assets, err := scrapeUnrealSource(); err != nil {
		log.Printf("Unreal Source error: %v", err)
	} else {
		for _, a := range assets {
			if _, seen := appData.SeenAssets[a.URL]; !seen {
				a.FirstSeen = time.Now()
				appData.SeenAssets[a.URL] = a
				newAssets = append(newAssets, a)
			}
		}
	}

	appData.LastCheck = time.Now()
	saveData()
	refreshAssetsList()

	if len(newAssets) > 0 {
		notifyNewAssets(newAssets)
	}
	log.Printf("Check complete. Found %d new assets.", len(newAssets))
}

func scrapeFabFreeAssets() ([]Asset, error) {
	req, _ := http.NewRequest("GET", fabFreeURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var assets []Asset
	doc.Find("a[href*='/listings/']").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		title := strings.TrimSpace(s.Text())
		if title == "" {
			title = s.Find("img").AttrOr("alt", "Unknown")
		}
		if href != "" && title != "" {
			if !strings.HasPrefix(href, "http") {
				href = "https://www.fab.com" + href
			}
			assets = append(assets, Asset{Title: title, URL: href, Price: "Free"})
		}
	})
	return assets, nil
}

func scrapeUnrealFreeAssets() ([]Asset, error) {
	req, _ := http.NewRequest("GET", unrealFreeURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var assets []Asset
	doc.Find("a[href*='/product/']").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		title := strings.TrimSpace(s.Text())
		if title == "" {
			title = s.Find("img").AttrOr("alt", "")
		}
		if href != "" && title != "" {
			if !strings.HasPrefix(href, "http") {
				href = "https://www.unrealengine.com" + href
			}
			assets = append(assets, Asset{Title: title, URL: href, Price: "Free"})
		}
	})
	return assets, nil
}

func scrapeUnrealSource() ([]Asset, error) {
	req, _ := http.NewRequest("GET", unrealSourceURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var assets []Asset
	doc.Find("a[href*='fab.com/listings']").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		title := strings.TrimSpace(s.Text())
		if title == "" {
			title = s.Parent().Text()
		}
		if len(title) > 100 {
			title = title[:100]
		}
		if href != "" && title != "" {
			assets = append(assets, Asset{Title: title, URL: href, Price: "Free"})
		}
	})
	return assets, nil
}

func notifyNewAssets(assets []Asset) {
	msg := fmt.Sprintf("%d new free assets!", len(assets))
	if len(assets) == 1 {
		msg = "New: " + assets[0].Title
	} else if len(assets) <= 3 {
		var titles []string
		for _, a := range assets {
			titles = append(titles, a.Title)
		}
		msg = strings.Join(titles, "\n")
	}

	notification := toast.Notification{
		AppID:   "Unreal Free Assets",
		Title:   "New Free Assets Found!",
		Message: msg,
	}
	notification.Push()
}

func clearHistory() {
	appData.SeenAssets = make(map[string]Asset)
	saveData()
	log.Println("History cleared")
}

func loadData() {
	appData.SeenAssets = make(map[string]Asset)
	data, err := os.ReadFile(dataFile)
	if err == nil {
		json.Unmarshal(data, &appData)
	}
}

func saveData() {
	data, _ := json.MarshalIndent(appData, "", "  ")
	os.WriteFile(dataFile, data, 0644)
}

func openBrowser(url string) {
	exec.Command("cmd", "/c", "start", "", url).Start()
}
