package sftp

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"vshell/internal/crypto"
	"vshell/internal/models"
)

type Direction string

const (
	DirUpload   Direction = "upload"
	DirDownload Direction = "download"
)

type TransferProgress struct {
	ID          string    `json:"id"`
	FileName    string    `json:"file_name"`
	TotalBytes  int64     `json:"total_bytes"`
	Transferred int64     `json:"transferred"`
	Percent     float64   `json:"percent"`
	SpeedKBps   float64   `json:"speed_kbps"`
	Done        bool      `json:"done"`
	Error       string    `json:"error,omitempty"`
	Direction   Direction `json:"direction"`
}

type Client struct {
	sftpClient *sftp.Client
	crypto     *crypto.CryptoService
	onEvent    func(string, any)
	mu         sync.Mutex
}

func NewClient(sshClient *ssh.Client, cryptoSvc *crypto.CryptoService, onEvent func(string, any)) (*Client, error) {
	sc, err := sftp.NewClient(sshClient)
	if err != nil {
		return nil, fmt.Errorf("sftp client: %w", err)
	}

	return &Client{
		sftpClient: sc,
		crypto:     cryptoSvc,
		onEvent:    onEvent,
	}, nil
}

func (c *Client) Close() error {
	return c.sftpClient.Close()
}

func (c *Client) Upload(localPath, remotePath string) error {
	stat, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("stat %s: %w", localPath, err)
	}
	if stat.IsDir() {
		return c.uploadDir(localPath, remotePath)
	}
	return c.UploadFile(localPath, remotePath)
}

// localTotalSize walks a local directory and returns the total file size.
func localTotalSize(dir string) (int64, error) {
	var total int64
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return total, err
}

// remoteTotalSize walks a remote directory and returns the total file size.
func (c *Client) remoteTotalSize(dir string) (int64, error) {
	var total int64
	entries, err := c.sftpClient.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	for _, e := range entries {
		path := dir + "/" + e.Name()
		if e.IsDir() {
			s, err := c.remoteTotalSize(path)
			if err != nil {
				return 0, err
			}
			total += s
		} else {
			total += e.Size()
		}
	}
	return total, nil
}

func (c *Client) uploadDir(localDir, remoteDir string) error {
	totalSize, err := localTotalSize(localDir)
	if err != nil {
		return fmt.Errorf("scan local dir: %w", err)
	}
	id := uuid.New().String()
	dirName := filepath.Base(localDir)
	start := time.Now()
	dp := &dirProgress{onEvent: c.onEvent, total: totalSize, startTime: start, id: id, fileName: dirName, direction: DirUpload}
	err = c.uploadDirRecursive(localDir, remoteDir, id, dirName, totalSize, dp)
	pct := float64(100)
	dp.onEvent("sftp:progress", TransferProgress{
		ID:          dp.id,
		FileName:    dp.fileName,
		TotalBytes:  dp.total,
		Transferred: dp.transferred,
		Percent:     pct,
		SpeedKBps:   float64(dp.transferred) / 1024 / time.Since(dp.startTime).Seconds(),
		Done:        true,
		Error: func() string {
			if err != nil {
				return err.Error()
			}
			return ""
		}(),
		Direction: dp.direction,
	})
	return err
}

type dirProgress struct {
	onEvent     func(string, any)
	total       int64
	transferred int64
	startTime   time.Time
	id          string
	fileName    string
	direction   Direction
}

func (d *dirProgress) add(n int64, currentFile string) {
	d.transferred += n
	elapsed := time.Since(d.startTime).Seconds()
	if elapsed < 0.1 {
		elapsed = 0.1
	}
	pct := float64(d.transferred) / float64(d.total) * 100
	if pct > 100 {
		pct = 100
	}
	d.onEvent("sftp:progress", TransferProgress{
		ID:          d.id,
		FileName:    currentFile,
		TotalBytes:  d.total,
		Transferred: d.transferred,
		Percent:     pct,
		SpeedKBps:   float64(d.transferred) / 1024 / elapsed,
		Done:        false,
		Direction:   d.direction,
	})
}

func (c *Client) uploadDirRecursive(localDir, remoteDir string, id string, dirName string, totalSize int64, dp *dirProgress) error {
	if err := c.sftpClient.MkdirAll(remoteDir); err != nil {
		return fmt.Errorf("mkdir remote %s: %w", remoteDir, err)
	}
	entries, err := os.ReadDir(localDir)
	if err != nil {
		return fmt.Errorf("read local dir %s: %w", localDir, err)
	}
	for _, entry := range entries {
		localPath := filepath.Join(localDir, entry.Name())
		remotePath := remoteDir + "/" + entry.Name()
		if entry.IsDir() {
			if err := c.uploadDirRecursive(localPath, remotePath, id, dirName, totalSize, dp); err != nil {
				return err
			}
		} else {
			if err := c.uploadSingleFile(localPath, remotePath, dp); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Client) uploadSingleFile(localPath, remotePath string, dp *dirProgress) error {
	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open local file: %w", err)
	}
	defer f.Close()

	remoteFile, err := c.sftpClient.Create(remotePath)
	if err != nil {
		return fmt.Errorf("create remote file: %w", err)
	}
	defer remoteFile.Close()

	buf := make([]byte, 32*1024)
	for {
		n, readErr := f.Read(buf)
		if n > 0 {
			if _, writeErr := remoteFile.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			dp.add(int64(n), filepath.Base(localPath))
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	return nil
}

func (c *Client) Download(remotePath, localPath string) error {
	stat, err := c.sftpClient.Stat(remotePath)
	if err != nil {
		return fmt.Errorf("stat %s: %w", remotePath, err)
	}
	if stat.IsDir() {
		return c.downloadDir(remotePath, localPath)
	}
	return c.DownloadFile(remotePath, localPath)
}

func (c *Client) downloadDir(remoteDir, localDir string) error {
	totalSize, err := c.remoteTotalSize(remoteDir)
	if err != nil {
		return fmt.Errorf("scan remote dir: %w", err)
	}
	id := uuid.New().String()
	dirName := filepath.Base(remoteDir)
	start := time.Now()
	dp := &dirProgress{onEvent: c.onEvent, total: totalSize, startTime: start, id: id, fileName: dirName, direction: DirDownload}
	err = c.downloadDirRecursive(remoteDir, localDir, dp)
	// emit final done event
	pct := float64(100)
	dp.onEvent("sftp:progress", TransferProgress{
		ID:          dp.id,
		FileName:    dp.fileName,
		TotalBytes:  dp.total,
		Transferred: dp.transferred,
		Percent:     pct,
		SpeedKBps:   float64(dp.transferred) / 1024 / time.Since(dp.startTime).Seconds(),
		Done:        true,
		Error: func() string {
			if err != nil {
				return err.Error()
			}
			return ""
		}(),
		Direction: dp.direction,
	})
	return err
}

func (c *Client) downloadDirRecursive(remoteDir, localDir string, dp *dirProgress) error {
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return fmt.Errorf("mkdir local %s: %w", localDir, err)
	}
	entries, err := c.sftpClient.ReadDir(remoteDir)
	if err != nil {
		return fmt.Errorf("read remote dir %s: %w", remoteDir, err)
	}
	for _, entry := range entries {
		remotePath := remoteDir + "/" + entry.Name()
		localPath := filepath.Join(localDir, entry.Name())
		if entry.IsDir() {
			if err := c.downloadDirRecursive(remotePath, localPath, dp); err != nil {
				return err
			}
		} else {
			if err := c.downloadSingleFile(remotePath, localPath, dp); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Client) downloadSingleFile(remotePath, localPath string, dp *dirProgress) error {
	remoteFile, err := c.sftpClient.Open(remotePath)
	if err != nil {
		return fmt.Errorf("open remote file: %w", err)
	}
	defer remoteFile.Close()

	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("create local dir: %w", err)
	}

	localFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("create local file: %w", err)
	}
	defer localFile.Close()

	buf := make([]byte, 32*1024)
	for {
		n, readErr := remoteFile.Read(buf)
		if n > 0 {
			if _, writeErr := localFile.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			dp.add(int64(n), filepath.Base(remotePath))
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	return nil
}

func (c *Client) UploadFile(localPath, remotePath string) error {
	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open local file: %w", err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat local file: %w", err)
	}

	remoteFile, err := c.sftpClient.Create(remotePath)
	if err != nil {
		return fmt.Errorf("create remote file: %w", err)
	}
	defer remoteFile.Close()

	id := uuid.New().String()
	pr := &progressReader{
		reader:     f,
		total:      stat.Size(),
		transferID: id,
		fileName:   filepath.Base(localPath),
		direction:  DirUpload,
		onEvent:    c.onEvent,
		startTime:  time.Now(),
	}

	_, err = io.Copy(remoteFile, pr)

	c.onEvent("sftp:progress", TransferProgress{
		ID:          id,
		FileName:    filepath.Base(localPath),
		TotalBytes:  stat.Size(),
		Transferred: pr.read,
		Percent:     100,
		Done:        true,
		Error: func() string {
			if err != nil {
				return err.Error()
			}
			return ""
		}(),
		Direction: DirUpload,
	})

	return err
}

func (c *Client) DownloadFile(remotePath, localPath string) error {
	remoteFile, err := c.sftpClient.Open(remotePath)
	if err != nil {
		return fmt.Errorf("open remote file: %w", err)
	}
	defer remoteFile.Close()

	stat, err := remoteFile.Stat()
	if err != nil {
		return fmt.Errorf("stat remote file: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("create local dir: %w", err)
	}

	localFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("create local file: %w", err)
	}
	defer localFile.Close()

	id := uuid.New().String()
	pw := &progressWriter{
		writer:     localFile,
		total:      stat.Size(),
		transferID: id,
		fileName:   filepath.Base(remotePath),
		direction:  DirDownload,
		onEvent:    c.onEvent,
		startTime:  time.Now(),
	}

	_, err = io.Copy(pw, remoteFile)

	c.onEvent("sftp:progress", TransferProgress{
		ID:          id,
		FileName:    filepath.Base(remotePath),
		TotalBytes:  stat.Size(),
		Transferred: pw.written,
		Percent:     100,
		Done:        true,
		Error: func() string {
			if err != nil {
				return err.Error()
			}
			return ""
		}(),
		Direction: DirDownload,
	})

	return err
}

func (c *Client) ReadDir(path string) ([]FileInfo, error) {
	entries, err := c.sftpClient.ReadDir(path)
	if err != nil {
		return nil, err
	}

	result := make([]FileInfo, len(entries))
	for i, e := range entries {
		result[i] = FileInfo{
			Name:    e.Name(),
			Size:    e.Size(),
			Mode:    uint32(e.Mode()),
			ModTime: e.ModTime().Unix(),
			IsDir:   e.IsDir(),
		}
	}
	return result, nil
}

func (c *Client) Remove(path string) error {
	stat, err := c.sftpClient.Stat(path)
	if err != nil {
		return fmt.Errorf("stat %s: %w", path, err)
	}
	if stat.IsDir() {
		return c.removeDirRecursive(path)
	}
	return c.sftpClient.Remove(path)
}

func (c *Client) removeDirRecursive(dir string) error {
	entries, err := c.sftpClient.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		path := dir + "/" + e.Name()
		if e.IsDir() {
			if err := c.removeDirRecursive(path); err != nil {
				return err
			}
		} else {
			if err := c.sftpClient.Remove(path); err != nil {
				return err
			}
		}
	}
	return c.sftpClient.RemoveDirectory(dir)
}

type FileInfo struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	Mode    uint32 `json:"mode"`
	ModTime int64  `json:"mod_time"`
	IsDir   bool   `json:"is_dir"`
}

// progressReader wraps io.Reader to track transfer progress.
type progressReader struct {
	reader     io.Reader
	total      int64
	read       int64
	transferID string
	fileName   string
	direction  Direction
	onEvent    func(string, any)
	startTime  time.Time
}

func (r *progressReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.read += int64(n)

	elapsed := time.Since(r.startTime).Seconds()
	if elapsed < 0.1 {
		elapsed = 0.1
	}
	percent := float64(r.read) / float64(r.total) * 100

	r.onEvent("sftp:progress", TransferProgress{
		ID:          r.transferID,
		FileName:    r.fileName,
		TotalBytes:  r.total,
		Transferred: r.read,
		Percent:     percent,
		SpeedKBps:   float64(r.read) / 1024 / elapsed,
		Done:        err == io.EOF,
		Direction:   r.direction,
	})

	return n, err
}

// progressWriter wraps io.Writer to track transfer progress.
type progressWriter struct {
	writer     io.Writer
	total      int64
	written    int64
	transferID string
	fileName   string
	direction  Direction
	onEvent    func(string, any)
	startTime  time.Time
}

func (w *progressWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	w.written += int64(n)

	elapsed := time.Since(w.startTime).Seconds()
	if elapsed < 0.1 {
		elapsed = 0.1
	}
	percent := float64(w.written) / float64(w.total) * 100

	w.onEvent("sftp:progress", TransferProgress{
		ID:          w.transferID,
		FileName:    w.fileName,
		TotalBytes:  w.total,
		Transferred: w.written,
		Percent:     percent,
		SpeedKBps:   float64(w.written) / 1024 / elapsed,
		Done:        false,
		Direction:   w.direction,
	})

	return n, err
}

// Pool manages concurrent file transfers.
type Pool struct {
	mu     sync.Mutex
	sem    chan struct{}
	active map[string]bool
}

func NewPool(maxConcurrent int) *Pool {
	return &Pool{
		sem:    make(chan struct{}, maxConcurrent),
		active: make(map[string]bool),
	}
}

func (p *Pool) Acquire(id string) bool {
	p.sem <- struct{}{}
	p.mu.Lock()
	p.active[id] = true
	p.mu.Unlock()
	return true
}

func (p *Pool) Release(id string) {
	p.mu.Lock()
	delete(p.active, id)
	p.mu.Unlock()
	<-p.sem
}

// BuildSFTPConfig creates SFTP client config from connection model (placeholder).
func BuildSFTPConfig(conn *models.Connection, cryptoSvc *crypto.CryptoService) (*ssh.ClientConfig, error) {
	_ = conn
	_ = cryptoSvc
	return nil, fmt.Errorf("not yet implemented: use ssh.Manager to get the existing client")
}
