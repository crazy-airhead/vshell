package sftp

import (
	"fmt"
	"io"
	"sync"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"vshell/internal/crypto"
	"vshell/internal/models"
)

type TransferProgress struct {
	ID          string  `json:"id"`
	FileName    string  `json:"file_name"`
	TotalBytes  int64   `json:"total_bytes"`
	Transferred int64   `json:"transferred"`
	Percent     float64 `json:"percent"`
	SpeedKBps   float64 `json:"speed_kbps"`
	Done        bool    `json:"done"`
	Error       string  `json:"error,omitempty"`
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
	onEvent    func(string, any)
	startTime  int64 // unix milliseconds
}

func (r *progressReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.read += int64(n)

	elapsed := float64(1) // avoid division by zero
	percent := float64(r.read) / float64(r.total) * 100

	r.onEvent("sftp:progress", TransferProgress{
		ID:          r.transferID,
		FileName:    r.fileName,
		TotalBytes:  r.total,
		Transferred: r.read,
		Percent:     percent,
		SpeedKBps:   float64(r.read) / 1024 / elapsed,
		Done:        err == io.EOF,
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
	onEvent    func(string, any)
}

func (w *progressWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	w.written += int64(n)

	percent := float64(w.written) / float64(w.total) * 100

	w.onEvent("sftp:progress", TransferProgress{
		ID:          w.transferID,
		FileName:    w.fileName,
		TotalBytes:  w.total,
		Transferred: w.written,
		Percent:     percent,
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
