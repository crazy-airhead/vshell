package app

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/pbkdf2"

	"vshell/internal/models"
)

const connectionExportVersion = 1
const connectionExportKDFIterations = 200000

type ConnectionImportResult struct {
	Groups      int `json:"groups"`
	Connections int `json:"connections"`
}

type connectionExportFile struct {
	Version     int                          `json:"version"`
	Encrypted   bool                         `json:"encrypted,omitempty"`
	ExportedAt  time.Time                    `json:"exported_at"`
	Groups      []connectionExportGroup      `json:"groups"`
	Connections []connectionExportConnection `json:"connections"`
}

type encryptedConnectionExportFile struct {
	Version    int       `json:"version"`
	Encrypted  bool      `json:"encrypted"`
	ExportedAt time.Time `json:"exported_at"`
	KDF        string    `json:"kdf"`
	Iterations int       `json:"iterations"`
	Salt       string    `json:"salt"`
	Nonce      string    `json:"nonce"`
	Data       string    `json:"data"`
}

type connectionExportGroup struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	ParentID  *string `json:"parent_id"`
	SortOrder int     `json:"sort_order"`
}

type connectionExportConnection struct {
	ID            string          `json:"id"`
	GroupID       *string         `json:"group_id"`
	Name          string          `json:"name"`
	Host          string          `json:"host"`
	Port          int             `json:"port"`
	Username      string          `json:"username"`
	AuthType      models.AuthType `json:"auth_type"`
	Password      string          `json:"password,omitempty"`
	PrivateKey    string          `json:"private_key,omitempty"`
	KeyPassphrase string          `json:"key_passphrase,omitempty"`
	ProxyType     *string         `json:"proxy_type"`
	ProxyAddr     *string         `json:"proxy_addr"`
	JumpHostID    *string         `json:"jump_host_id"`
	UploadPath    string          `json:"upload_path"`
	DefaultCmd    *string         `json:"default_cmd"`
	SortOrder     int             `json:"sort_order"`
	Color         *string         `json:"color"`
}

func (a *AppService) ExportConnectionConfigs(filePath string, password string) error {
	if filePath == "" {
		return nil
	}

	exportFile := connectionExportFile{
		Version:    connectionExportVersion,
		ExportedAt: time.Now(),
	}

	groups, err := a.exportGroups()
	if err != nil {
		return err
	}
	exportFile.Groups = groups

	connections, err := a.exportConnections()
	if err != nil {
		return err
	}
	exportFile.Connections = connections

	data, err := json.MarshalIndent(exportFile, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal export: %w", err)
	}
	if password != "" {
		data, err = encryptConnectionExport(data, password, exportFile.ExportedAt)
		if err != nil {
			return err
		}
	}

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("create export directory: %w", err)
	}
	return os.WriteFile(filePath, data, 0600)
}

func (a *AppService) IsConnectionConfigEncrypted(filePath string) (bool, error) {
	if filePath == "" {
		return false, nil
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false, fmt.Errorf("read import file: %w", err)
	}
	return isEncryptedConnectionExport(data)
}

func (a *AppService) ImportConnectionConfigs(filePath string, password string) (ConnectionImportResult, error) {
	var result ConnectionImportResult
	if filePath == "" {
		return result, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return result, fmt.Errorf("read import file: %w", err)
	}
	if encrypted, err := isEncryptedConnectionExport(data); err != nil {
		return result, err
	} else if encrypted {
		data, err = decryptConnectionExport(data, password)
		if err != nil {
			return result, err
		}
	}

	var importFile connectionExportFile
	if err := json.Unmarshal(data, &importFile); err != nil {
		return result, fmt.Errorf("parse import file: %w", err)
	}
	if importFile.Version != connectionExportVersion {
		return result, fmt.Errorf("unsupported import version: %d", importFile.Version)
	}

	tx, err := a.db.Begin()
	if err != nil {
		return result, err
	}
	defer tx.Rollback()

	for _, group := range importFile.Groups {
		if group.ID == "" || group.Name == "" {
			continue
		}
		if _, err := tx.Exec(
			`INSERT INTO groups (id, name, parent_id, sort_order)
			 VALUES (?, ?, NULL, ?)
			 ON CONFLICT(id) DO UPDATE SET name=excluded.name, sort_order=excluded.sort_order, updated_at=CURRENT_TIMESTAMP`,
			group.ID, group.Name, group.SortOrder,
		); err != nil {
			return result, fmt.Errorf("import group %s: %w", group.Name, err)
		}
		result.Groups++
	}

	for _, group := range importFile.Groups {
		if group.ID == "" {
			continue
		}
		if _, err := tx.Exec("UPDATE groups SET parent_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", group.ParentID, group.ID); err != nil {
			return result, fmt.Errorf("restore group parent %s: %w", group.ID, err)
		}
	}

	for _, conn := range importFile.Connections {
		if conn.ID == "" || conn.Name == "" || conn.Host == "" {
			continue
		}
		if conn.Port == 0 {
			conn.Port = 22
		}
		if conn.UploadPath == "" {
			conn.UploadPath = "/"
		}
		groupID, err := existingGroupID(tx, conn.GroupID)
		if err != nil {
			return result, err
		}
		password, err := a.db.Crypto().Encrypt(conn.Password)
		if err != nil {
			return result, fmt.Errorf("encrypt password for %s: %w", conn.Name, err)
		}
		privateKey, err := a.db.Crypto().Encrypt(conn.PrivateKey)
		if err != nil {
			return result, fmt.Errorf("encrypt private key for %s: %w", conn.Name, err)
		}
		keyPassphrase, err := a.db.Crypto().Encrypt(conn.KeyPassphrase)
		if err != nil {
			return result, fmt.Errorf("encrypt key passphrase for %s: %w", conn.Name, err)
		}

		if _, err := tx.Exec(
			`INSERT INTO connections (id, group_id, name, host, port, username, auth_type, password, private_key, key_passphrase, proxy_type, proxy_addr, jump_host_id, upload_path, default_cmd, sort_order, color)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			 ON CONFLICT(id) DO UPDATE SET
			   group_id=excluded.group_id,
			   name=excluded.name,
			   host=excluded.host,
			   port=excluded.port,
			   username=excluded.username,
			   auth_type=excluded.auth_type,
			   password=excluded.password,
			   private_key=excluded.private_key,
			   key_passphrase=excluded.key_passphrase,
			   proxy_type=excluded.proxy_type,
			   proxy_addr=excluded.proxy_addr,
			   jump_host_id=excluded.jump_host_id,
			   upload_path=excluded.upload_path,
			   default_cmd=excluded.default_cmd,
			   sort_order=excluded.sort_order,
			   color=excluded.color,
			   updated_at=CURRENT_TIMESTAMP`,
			conn.ID, groupID, conn.Name, conn.Host, conn.Port, conn.Username, conn.AuthType,
			password, privateKey, keyPassphrase, conn.ProxyType, conn.ProxyAddr, conn.JumpHostID,
			conn.UploadPath, conn.DefaultCmd, conn.SortOrder, conn.Color,
		); err != nil {
			return result, fmt.Errorf("import connection %s: %w", conn.Name, err)
		}
		result.Connections++
	}

	if err := tx.Commit(); err != nil {
		return result, err
	}
	return result, nil
}

func (a *AppService) exportGroups() ([]connectionExportGroup, error) {
	rows, err := a.db.Query("SELECT id, name, parent_id, sort_order FROM groups ORDER BY sort_order, name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []connectionExportGroup
	for rows.Next() {
		var group connectionExportGroup
		if err := rows.Scan(&group.ID, &group.Name, &group.ParentID, &group.SortOrder); err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	return groups, rows.Err()
}

func (a *AppService) exportConnections() ([]connectionExportConnection, error) {
	rows, err := a.db.Query(`
		SELECT id, group_id, name, host, port, username, auth_type, password, private_key, key_passphrase,
		       proxy_type, proxy_addr, jump_host_id, upload_path, default_cmd, sort_order, color
		FROM connections
		ORDER BY sort_order, name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var connections []connectionExportConnection
	for rows.Next() {
		var conn connectionExportConnection
		if err := rows.Scan(
			&conn.ID, &conn.GroupID, &conn.Name, &conn.Host, &conn.Port, &conn.Username, &conn.AuthType,
			&conn.Password, &conn.PrivateKey, &conn.KeyPassphrase, &conn.ProxyType, &conn.ProxyAddr,
			&conn.JumpHostID, &conn.UploadPath, &conn.DefaultCmd, &conn.SortOrder, &conn.Color,
		); err != nil {
			return nil, err
		}

		var err error
		if conn.Password, err = a.db.Crypto().Decrypt(conn.Password); err != nil {
			return nil, fmt.Errorf("decrypt password for %s: %w", conn.Name, err)
		}
		if conn.PrivateKey, err = a.db.Crypto().Decrypt(conn.PrivateKey); err != nil {
			return nil, fmt.Errorf("decrypt private key for %s: %w", conn.Name, err)
		}
		if conn.KeyPassphrase, err = a.db.Crypto().Decrypt(conn.KeyPassphrase); err != nil {
			return nil, fmt.Errorf("decrypt key passphrase for %s: %w", conn.Name, err)
		}

		connections = append(connections, conn)
	}
	return connections, rows.Err()
}

func existingGroupID(tx *sql.Tx, groupID *string) (*string, error) {
	if groupID == nil || *groupID == "" {
		return nil, nil
	}
	var id string
	if err := tx.QueryRow("SELECT id FROM groups WHERE id = ?", *groupID).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &id, nil
}

func isEncryptedConnectionExport(data []byte) (bool, error) {
	var meta struct {
		Version   int  `json:"version"`
		Encrypted bool `json:"encrypted"`
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return false, fmt.Errorf("parse import file: %w", err)
	}
	return meta.Encrypted, nil
}

func encryptConnectionExport(plaintext []byte, password string, exportedAt time.Time) ([]byte, error) {
	salt := make([]byte, 16)
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	aead, err := connectionExportAEAD(password, salt, connectionExportKDFIterations)
	if err != nil {
		return nil, err
	}
	encrypted := encryptedConnectionExportFile{
		Version:    connectionExportVersion,
		Encrypted:  true,
		ExportedAt: exportedAt,
		KDF:        "PBKDF2-SHA256",
		Iterations: connectionExportKDFIterations,
		Salt:       base64.StdEncoding.EncodeToString(salt),
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Data:       base64.StdEncoding.EncodeToString(aead.Seal(nil, nonce, plaintext, nil)),
	}
	return json.MarshalIndent(encrypted, "", "  ")
}

func decryptConnectionExport(data []byte, password string) ([]byte, error) {
	if password == "" {
		return nil, fmt.Errorf("password is required for encrypted import")
	}
	var encrypted encryptedConnectionExportFile
	if err := json.Unmarshal(data, &encrypted); err != nil {
		return nil, fmt.Errorf("parse encrypted import file: %w", err)
	}
	if encrypted.Version != connectionExportVersion {
		return nil, fmt.Errorf("unsupported import version: %d", encrypted.Version)
	}
	if encrypted.KDF != "PBKDF2-SHA256" {
		return nil, fmt.Errorf("unsupported import KDF: %s", encrypted.KDF)
	}
	salt, err := base64.StdEncoding.DecodeString(encrypted.Salt)
	if err != nil {
		return nil, fmt.Errorf("decode salt: %w", err)
	}
	nonce, err := base64.StdEncoding.DecodeString(encrypted.Nonce)
	if err != nil {
		return nil, fmt.Errorf("decode nonce: %w", err)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted.Data)
	if err != nil {
		return nil, fmt.Errorf("decode encrypted data: %w", err)
	}
	aead, err := connectionExportAEAD(password, salt, encrypted.Iterations)
	if err != nil {
		return nil, err
	}
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt import file: invalid password or corrupted file")
	}
	return plaintext, nil
}

func connectionExportAEAD(password string, salt []byte, iterations int) (cipher.AEAD, error) {
	if iterations <= 0 {
		iterations = connectionExportKDFIterations
	}
	key := pbkdf2.Key([]byte(password), salt, iterations, 32, sha256.New)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
