package app

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/oschwald/geoip2-golang"
)

var geoIPDB struct {
	sync.Mutex
	path   string
	reader *geoip2.Reader
}

const defaultGeoIPDownloadURL = "https://raw.githubusercontent.com/Loyalsoldier/geoip/release/Country-without-asn.mmdb"

type geoIPSettings struct {
	DownloadURL string `json:"download_url"`
}

var geoIPDownloadMu sync.Mutex

func (a *AppService) LookupIPCountry(host string) (string, error) {
	ip := net.ParseIP(strings.TrimSpace(host))
	if ip == nil || isPrivateIP(ip) {
		return "", nil
	}

	reader, err := getGeoIPReader()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	if reader == nil {
		return "", nil
	}

	record, err := reader.Country(ip)
	if err != nil {
		return "", fmt.Errorf("lookup country: %w", err)
	}

	country := strings.ToLower(record.Country.IsoCode)
	if country == "" && record.RegisteredCountry.IsoCode != "" {
		country = strings.ToLower(record.RegisteredCountry.IsoCode)
	}
	return country, nil
}

func (a *AppService) ensureGeoIPDatabase(emit func(string, any)) {
	path, err := findGeoIPDatabase()
	if err != nil || path != "" {
		return
	}

	url := geoIPDownloadURL()
	if url == "" {
		return
	}

	emit("geoip:download:start", nil)
	if err := downloadGeoIPDatabase(url); err != nil {
		emit("geoip:download:error", err.Error())
		return
	}
	emit("geoip:download:done", nil)
}

func (a *AppService) GetGeoIPDownloadURL() (string, error) {
	return geoIPDownloadURL(), nil
}

func (a *AppService) SetGeoIPDownloadURL(url string) error {
	url = strings.TrimSpace(url)
	if url == "" {
		url = defaultGeoIPDownloadURL
	}
	return saveGeoIPSettings(geoIPSettings{DownloadURL: url})
}

func (a *AppService) UpdateGeoIPDatabase() error {
	emit := func(event string, data any) {
		if a.wailsApp != nil {
			a.wailsApp.Event.Emit(event, data)
		}
	}

	url := geoIPDownloadURL()
	emit("geoip:download:start", nil)
	if err := downloadGeoIPDatabase(url); err != nil {
		emit("geoip:download:error", err.Error())
		return err
	}
	emit("geoip:download:done", nil)
	return nil
}

func getGeoIPReader() (*geoip2.Reader, error) {
	path, err := findGeoIPDatabase()
	if err != nil {
		return nil, err
	}
	if path == "" {
		return nil, nil
	}

	geoIPDB.Lock()
	defer geoIPDB.Unlock()

	if geoIPDB.reader != nil && geoIPDB.path == path {
		return geoIPDB.reader, nil
	}

	if geoIPDB.reader != nil {
		geoIPDB.reader.Close()
		geoIPDB.reader = nil
	}

	reader, err := geoip2.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open geoip database: %w", err)
	}
	geoIPDB.path = path
	geoIPDB.reader = reader
	return reader, nil
}

func findGeoIPDatabase() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("get user config dir: %w", err)
	}

	candidates := []string{
		filepath.Join(configDir, "vshell", "GeoLite2-Country.mmdb"),
		filepath.Join(configDir, "vshell", "geoip", "GeoLite2-Country.mmdb"),
		filepath.Join(configDir, "vshell", "GeoIP2-Country.mmdb"),
	}

	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "GeoLite2-Country.mmdb"),
			filepath.Join(cwd, "geoip", "GeoLite2-Country.mmdb"),
			filepath.Join(cwd, "frontend", "public", "geoip", "GeoLite2-Country.mmdb"),
		)
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
	}
	return "", nil
}

func geoIPDownloadURL() string {
	if url := strings.TrimSpace(os.Getenv("VSHELL_GEOIP_DOWNLOAD_URL")); url != "" {
		return url
	}
	settings, err := loadGeoIPSettings()
	if err == nil && strings.TrimSpace(settings.DownloadURL) != "" {
		return strings.TrimSpace(settings.DownloadURL)
	}
	if key := strings.TrimSpace(os.Getenv("MAXMIND_LICENSE_KEY")); key != "" {
		return "https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-Country&license_key=" + key + "&suffix=tar.gz"
	}
	return defaultGeoIPDownloadURL
}

func downloadGeoIPDatabase(url string) error {
	geoIPDownloadMu.Lock()
	defer geoIPDownloadMu.Unlock()

	url = strings.TrimSpace(url)
	if url == "" {
		return fmt.Errorf("geoip download url is empty")
	}

	destPath, err := defaultGeoIPDatabasePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0700); err != nil {
		return fmt.Errorf("create geoip dir: %w", err)
	}

	tmpPath := destPath + ".download"
	defer os.Remove(tmpPath)

	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("download geoip database: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download geoip database: %s", resp.Status)
	}

	if strings.Contains(strings.ToLower(url), ".tar.gz") || strings.Contains(resp.Header.Get("Content-Type"), "gzip") {
		if err := extractMMDBFromTarGz(resp.Body, tmpPath); err != nil {
			return err
		}
	} else {
		file, err := os.Create(tmpPath)
		if err != nil {
			return fmt.Errorf("create geoip database: %w", err)
		}
		if _, err := io.Copy(file, resp.Body); err != nil {
			file.Close()
			return fmt.Errorf("write geoip database: %w", err)
		}
		if err := file.Close(); err != nil {
			return fmt.Errorf("close geoip database: %w", err)
		}
	}

	if _, err := geoip2.Open(tmpPath); err != nil {
		return fmt.Errorf("validate geoip database: %w", err)
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		return fmt.Errorf("install geoip database: %w", err)
	}
	resetGeoIPReader()
	return nil
}

func extractMMDBFromTarGz(reader io.Reader, destPath string) error {
	gz, err := gzip.NewReader(reader)
	if err != nil {
		return fmt.Errorf("open geoip archive: %w", err)
	}
	defer gz.Close()

	tarReader := tar.NewReader(gz)
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("read geoip archive: %w", err)
		}
		if header.Typeflag != tar.TypeReg || !strings.HasSuffix(header.Name, ".mmdb") {
			continue
		}

		file, err := os.Create(destPath)
		if err != nil {
			return fmt.Errorf("create geoip database: %w", err)
		}
		if _, err := io.Copy(file, tarReader); err != nil {
			file.Close()
			return fmt.Errorf("extract geoip database: %w", err)
		}
		if err := file.Close(); err != nil {
			return fmt.Errorf("close geoip database: %w", err)
		}
		return nil
	}
	return fmt.Errorf("geoip archive did not contain an mmdb file")
}

func defaultGeoIPDatabasePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("get user config dir: %w", err)
	}
	return filepath.Join(configDir, "vshell", "GeoLite2-Country.mmdb"), nil
}

func geoIPSettingsPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("get user config dir: %w", err)
	}
	return filepath.Join(configDir, "vshell", "geoip-settings.json"), nil
}

func loadGeoIPSettings() (geoIPSettings, error) {
	path, err := geoIPSettingsPath()
	if err != nil {
		return geoIPSettings{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return geoIPSettings{DownloadURL: defaultGeoIPDownloadURL}, nil
		}
		return geoIPSettings{}, err
	}

	var settings geoIPSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return geoIPSettings{}, fmt.Errorf("read geoip settings: %w", err)
	}
	if strings.TrimSpace(settings.DownloadURL) == "" {
		settings.DownloadURL = defaultGeoIPDownloadURL
	}
	return settings, nil
}

func saveGeoIPSettings(settings geoIPSettings) error {
	path, err := geoIPSettingsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create geoip settings dir: %w", err)
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal geoip settings: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}

func resetGeoIPReader() {
	geoIPDB.Lock()
	defer geoIPDB.Unlock()
	if geoIPDB.reader != nil {
		geoIPDB.reader.Close()
		geoIPDB.reader = nil
		geoIPDB.path = ""
	}
}

func isPrivateIP(ip net.IP) bool {
	return ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsUnspecified()
}
