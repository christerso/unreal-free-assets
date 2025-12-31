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
	"regexp"
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
	unrealSourceURL = "https://unrealsource.com/dispatch/"
	checkInterval   = 1 * time.Hour
	dataFileName    = "seen_assets.json"
)

// Asset categories
const (
	CategoryFree   = "free"
	CategoryLatest = "latest"
)

type Asset struct {
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	Price     string    `json:"price"`
	Category  string    `json:"category"` // "free" or "latest"
	ExpiresAt string    `json:"expires_at,omitempty"`
	FirstSeen time.Time `json:"first_seen"`
}

type AppData struct {
	SeenAssets map[string]Asset `json:"seen_assets"`
	LastCheck  time.Time        `json:"last_check"`
}

var (
	appData           AppData
	dataFile          string
	dataDir           string
	httpClient        *http.Client
	fyneApp           fyne.App
	mainWindow        fyne.Window
	freeList          *widget.List
	latestList        *widget.List
	freeAssets        []Asset
	latestAssets      []Asset
	filteredFree      []Asset
	filteredLatest    []Asset
	statusLabel       *widget.Label
	tabs              *container.AppTabs
	searchEntry       *widget.Entry
	currentSearchTerm string
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

	fyneApp = app.New()
	fyneApp.Settings().SetTheme(&unrealTheme{})

	if desk, ok := fyneApp.(desktop.App); ok {
		setupSystemTray(desk)
	}

	mainWindow = fyneApp.NewWindow("Unreal Assets Monitor")
	mainWindow.Resize(fyne.NewSize(800, 600))
	if iconData != nil && len(iconData) > 0 {
		mainWindow.SetIcon(fyne.NewStaticResource("icon.png", iconData))
	}
	mainWindow.SetCloseIntercept(func() {
		mainWindow.Hide()
	})

	buildMainUI()

	go backgroundChecker()

	go func() {
		time.Sleep(3 * time.Second)
		checkForAssets()
	}()

	fyneApp.Run()
}

func setupSystemTray(desk desktop.App) {
	menu := fyne.NewMenu("Unreal Assets Monitor",
		fyne.NewMenuItem("View Assets", func() {
			refreshAssetLists()
			mainWindow.Show()
			mainWindow.RequestFocus()
		}),
		fyne.NewMenuItem("Check Now", func() {
			go checkForAssets()
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Open FAB Marketplace", func() {
			openBrowser("https://www.fab.com/search?price=free")
		}),
		fyne.NewMenuItem("Buy Me a Coffee", func() {
			openBrowser("https://buymeacoffee.com/qvark")
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
	title := canvas.NewText("Unreal Assets Monitor", color.RGBA{245, 130, 32, 255})
	title.TextSize = 26
	title.TextStyle = fyne.TextStyle{Bold: true}

	statusLabel = widget.NewLabel("Loading...")
	statusLabel.Alignment = fyne.TextAlignCenter

	// Search entry
	searchEntry = widget.NewEntry()
	searchEntry.SetPlaceHolder("Search assets...")
	searchEntry.OnChanged = func(s string) {
		currentSearchTerm = strings.ToLower(strings.TrimSpace(s))
		applySearchFilter()
	}

	// Clear search button
	clearSearchBtn := widget.NewButton("Clear", func() {
		searchEntry.SetText("")
		currentSearchTerm = ""
		applySearchFilter()
	})

	searchBox := container.NewBorder(nil, nil, nil, clearSearchBtn, searchEntry)

	header := container.NewVBox(
		container.NewCenter(title),
		statusLabel,
		widget.NewSeparator(),
		searchBox,
	)

	// Create lists - use filtered lists for display
	freeAssets, latestAssets = getSortedAssets()
	filteredFree = freeAssets
	filteredLatest = latestAssets

	// FREE tab
	freeList = createAssetList(&filteredFree)
	freeTab := container.NewBorder(
		createTabHeader("🎁 FREE Assets", "Claim these before they expire!", len(filteredFree)),
		nil, nil, nil,
		freeList,
	)

	// LATEST tab
	latestList = createAssetList(&filteredLatest)
	latestTab := container.NewBorder(
		createTabHeader("🆕 Latest News", "Unreal & FAB marketplace news", len(filteredLatest)),
		nil, nil, nil,
		latestList,
	)

	// Tabs
	tabs = container.NewAppTabs(
		container.NewTabItem(fmt.Sprintf("Free (%d)", len(filteredFree)), freeTab),
		container.NewTabItem(fmt.Sprintf("Latest (%d)", len(filteredLatest)), latestTab),
	)
	tabs.SetTabLocation(container.TabLocationTop)

	// Footer buttons
	checkBtn := widget.NewButton("🔄 Check Now", func() {
		go checkForAssets()
	})

	fabBtn := widget.NewButton("🌐 Open FAB", func() {
		openBrowser("https://www.fab.com/search?price=free")
	})
	fabBtn.Importance = widget.HighImportance

	clearBtn := widget.NewButton("🗑 Clear All", func() {
		clearHistory()
		refreshAssetLists()
	})

	coffeeBtn := widget.NewButton("☕ Buy Me a Coffee", func() {
		openBrowser("https://buymeacoffee.com/qvark")
	})

	footer := container.NewVBox(
		widget.NewSeparator(),
		container.NewCenter(container.NewHBox(checkBtn, fabBtn, clearBtn, coffeeBtn)),
	)

	mainWindow.SetContent(container.NewBorder(header, footer, nil, nil, tabs))
	updateStatusLabel()
}

func applySearchFilter() {
	if currentSearchTerm == "" {
		// No filter - show all
		filteredFree = freeAssets
		filteredLatest = latestAssets
	} else {
		// Filter by search term
		filteredFree = nil
		for _, a := range freeAssets {
			if strings.Contains(strings.ToLower(a.Title), currentSearchTerm) ||
				strings.Contains(strings.ToLower(a.URL), currentSearchTerm) {
				filteredFree = append(filteredFree, a)
			}
		}

		filteredLatest = nil
		for _, a := range latestAssets {
			if strings.Contains(strings.ToLower(a.Title), currentSearchTerm) ||
				strings.Contains(strings.ToLower(a.URL), currentSearchTerm) {
				filteredLatest = append(filteredLatest, a)
			}
		}
	}

	// Refresh lists
	if freeList != nil {
		freeList.Refresh()
	}
	if latestList != nil {
		latestList.Refresh()
	}

	// Update tab counts
	if tabs != nil {
		tabs.Items[0].Text = fmt.Sprintf("Free (%d)", len(filteredFree))
		tabs.Items[1].Text = fmt.Sprintf("Latest (%d)", len(filteredLatest))
		tabs.Refresh()
	}
}

func createTabHeader(title, subtitle string, count int) fyne.CanvasObject {
	titleText := canvas.NewText(title, color.RGBA{245, 130, 32, 255})
	titleText.TextSize = 18
	titleText.TextStyle = fyne.TextStyle{Bold: true}

	subtitleText := widget.NewLabel(subtitle)
	subtitleText.Alignment = fyne.TextAlignCenter

	return container.NewVBox(
		container.NewCenter(titleText),
		subtitleText,
		widget.NewSeparator(),
	)
}

func createAssetList(assets *[]Asset) *widget.List {
	list := widget.NewList(
		func() int { return len(*assets) },
		func() fyne.CanvasObject {
			titleLabel := widget.NewLabel("Asset Title Here")
			titleLabel.TextStyle = fyne.TextStyle{Bold: true}
			titleLabel.Wrapping = fyne.TextTruncate

			infoLabel := widget.NewLabel("Info")
			infoLabel.TextStyle = fyne.TextStyle{Italic: true}

			openBtn := widget.NewButton("Open", func() {})
			openBtn.Importance = widget.HighImportance

			left := container.NewVBox(titleLabel, infoLabel)
			return container.NewBorder(nil, nil, nil, openBtn, left)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(*assets) {
				return
			}
			asset := (*assets)[id]

			c := obj.(*fyne.Container)
			left := c.Objects[0].(*fyne.Container)
			titleLabel := left.Objects[0].(*widget.Label)
			infoLabel := left.Objects[1].(*widget.Label)
			openBtn := c.Objects[1].(*widget.Button)

			displayTitle := asset.Title
			if len(displayTitle) > 60 {
				displayTitle = displayTitle[:57] + "..."
			}
			titleLabel.SetText(displayTitle)

			// Show category-specific info
			if asset.Category == CategoryFree {
				if asset.ExpiresAt != "" {
					infoLabel.SetText("⏰ " + asset.ExpiresAt)
				} else {
					infoLabel.SetText("🎁 FREE - Claim now!")
				}
			} else {
				infoLabel.SetText("💰 " + asset.Price + " • Found: " + asset.FirstSeen.Format("Jan 2"))
			}

			url := asset.URL
			openBtn.OnTapped = func() { openBrowser(url) }
		},
	)

	list.OnSelected = func(id widget.ListItemID) {
		if id < len(*assets) {
			openBrowser((*assets)[id].URL)
		}
		list.Unselect(id)
	}

	return list
}

func refreshAssetLists() {
	freeAssets, latestAssets = getSortedAssets()
	// Re-apply current search filter
	applySearchFilter()
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
	total := len(freeAssets) + len(latestAssets)
	statusLabel.SetText(fmt.Sprintf("%d free • %d latest • Last check: %s", len(freeAssets), len(latestAssets), checkTime))
	_ = total
}

func getSortedAssets() ([]Asset, []Asset) {
	var free, latest []Asset
	for _, asset := range appData.SeenAssets {
		if asset.Category == CategoryFree {
			free = append(free, asset)
		} else {
			latest = append(latest, asset)
		}
	}
	sort.Slice(free, func(i, j int) bool {
		return free[i].FirstSeen.After(free[j].FirstSeen)
	})
	sort.Slice(latest, func(i, j int) bool {
		return latest[i].FirstSeen.After(latest[j].FirstSeen)
	})
	return free, latest
}

func backgroundChecker() {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()
	for range ticker.C {
		checkForAssets()
	}
}

func checkForAssets() {
	log.Println("Checking for assets...")
	newFreeAssets := []Asset{}
	newLatestAssets := []Asset{}

	// Scrape Unreal Source for FREE and Latest assets
	free, latest, err := scrapeUnrealSource()
	if err != nil {
		log.Printf("Unreal Source error: %v", err)
	} else {
		for _, a := range free {
			if _, seen := appData.SeenAssets[a.URL]; !seen {
				a.FirstSeen = time.Now()
				appData.SeenAssets[a.URL] = a
				newFreeAssets = append(newFreeAssets, a)
			}
		}
		for _, a := range latest {
			if _, seen := appData.SeenAssets[a.URL]; !seen {
				a.FirstSeen = time.Now()
				appData.SeenAssets[a.URL] = a
				newLatestAssets = append(newLatestAssets, a)
			}
		}
	}

	appData.LastCheck = time.Now()
	saveData()
	refreshAssetLists()

	if len(newFreeAssets) > 0 {
		notifyNewAssets(newFreeAssets, true)
	}
	if len(newLatestAssets) > 0 {
		notifyNewAssets(newLatestAssets, false)
	}

	log.Printf("Check complete. Found %d new free, %d new latest.", len(newFreeAssets), len(newLatestAssets))
}

func scrapeUnrealSource() ([]Asset, []Asset, error) {
	var freeAssets []Asset
	var latestAssets []Asset
	seenURLs := make(map[string]bool)

	// First, get the dispatch page to find links to free asset announcements
	req, _ := http.NewRequest("GET", unrealSourceURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	// Find links to "free fab assets" detail pages
	// These are dispatch articles with URLs like /d/free-fab-assets-*
	freeDispatchLinks := []string{}
	doc.Find("a[href*='/d/free-fab-assets']").Each(func(i int, link *goquery.Selection) {
		href, exists := link.Attr("href")
		if exists && !strings.HasPrefix(href, "http") {
			href = "https://unrealsource.com" + href
		}
		if exists && href != "" {
			// Dedupe
			for _, existing := range freeDispatchLinks {
				if existing == href {
					return
				}
			}
			freeDispatchLinks = append(freeDispatchLinks, href)
		}
	})

	// Fetch each free assets detail page (usually just 1 - the current batch)
	// Only process the first one (most recent)
	if len(freeDispatchLinks) > 0 {
		free := scrapeFreeAssetsPage(freeDispatchLinks[0], seenURLs)
		freeAssets = append(freeAssets, free...)
	}

	// Collect other dispatch links as "latest" news
	doc.Find("a[href*='/d/']").Each(func(i int, link *goquery.Selection) {
		href, exists := link.Attr("href")
		if !exists {
			return
		}
		// Skip free-fab-assets pages (already processed)
		if strings.Contains(href, "free-fab-assets") {
			return
		}
		if !strings.HasPrefix(href, "http") {
			href = "https://unrealsource.com" + href
		}
		if seenURLs[href] {
			return
		}

		// Try to get title from the link text, or extract from URL if it's a timestamp
		title := strings.TrimSpace(link.Text())

		// If title looks like a relative time, extract from URL instead
		if title == "" || len(title) < 5 || strings.Contains(title, "ago") ||
		   strings.Contains(title, "month") || strings.Contains(title, "year") ||
		   strings.Contains(title, "week") || strings.Contains(title, "day") {
			// Extract title from URL: /d/some-article-title/ -> Some Article Title
			parts := strings.Split(href, "/d/")
			if len(parts) > 1 {
				slug := strings.TrimSuffix(parts[1], "/")
				slug = strings.ReplaceAll(slug, "-", " ")
				// Capitalize first letter of each word
				words := strings.Fields(slug)
				for i, w := range words {
					if len(w) > 0 {
						words[i] = strings.ToUpper(string(w[0])) + w[1:]
					}
				}
				title = strings.Join(words, " ")
			}
		}

		if title == "" || len(title) < 5 {
			return
		}
		if len(title) > 80 {
			title = title[:77] + "..."
		}

		seenURLs[href] = true
		latestAssets = append(latestAssets, Asset{
			Title:    title,
			URL:      href,
			Price:    "News",
			Category: CategoryLatest,
		})
	})

	log.Printf("Scraped: %d free assets, %d latest assets", len(freeAssets), len(latestAssets))
	return freeAssets, latestAssets, nil
}

func scrapeFreeAssetsPage(url string, seenURLs map[string]bool) []Asset {
	var assets []Asset

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("Error fetching free assets page: %v", err)
		return assets
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("Free assets page returned status %d", resp.StatusCode)
		return assets
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Printf("Error parsing free assets page: %v", err)
		return assets
	}

	// Extract expiration date from page text
	expiresPattern := regexp.MustCompile(`(?i)(?:until|before).*?(\w+\s+\d+,?\s*202\d)`)
	expiresAt := ""
	pageText := doc.Text()
	if matches := expiresPattern.FindStringSubmatch(pageText); len(matches) > 1 {
		expiresAt = "Free until " + matches[1]
	}

	// Find all fab.com/listings links on this page - these are the free assets
	doc.Find("a[href*='fab.com/listings']").Each(func(i int, link *goquery.Selection) {
		href, exists := link.Attr("href")
		if !exists || seenURLs[href] {
			return
		}

		title := strings.TrimSpace(link.Text())
		if title == "" || len(title) < 3 {
			return
		}

		seenURLs[href] = true
		assets = append(assets, Asset{
			Title:     title,
			URL:       href,
			Price:     "FREE",
			Category:  CategoryFree,
			ExpiresAt: expiresAt,
		})
	})

	log.Printf("Found %d free assets on detail page", len(assets))
	return assets
}

func notifyNewAssets(assets []Asset, isFree bool) {
	title := "New Assets Found!"
	if isFree {
		title = "🎁 New FREE Assets!"
	}

	msg := fmt.Sprintf("%d new assets", len(assets))
	if len(assets) == 1 {
		msg = assets[0].Title
	} else if len(assets) <= 3 {
		var titles []string
		for _, a := range assets {
			titles = append(titles, a.Title)
		}
		msg = strings.Join(titles, "\n")
	}

	notification := toast.Notification{
		AppID:   "Unreal Assets Monitor",
		Title:   title,
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
