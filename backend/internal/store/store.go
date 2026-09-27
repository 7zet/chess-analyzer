package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ahror/royal-move/internal/domain"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

type Store struct{ DB *sql.DB }

func Open(ctx context.Context, dsn string) (*Store, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err = db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	s := &Store{DB: db}
	if err = s.migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate(ctx context.Context) error {
	for _, statement := range strings.Split(schema, ";") {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		if _, err := s.DB.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("database migration: %w", err)
		}
	}
	return nil
}

func (s *Store) CreateUser(ctx context.Context, email, name, hash, role string) (domain.User, error) {
	if role != "admin" {
		role = "user"
	}
	u := domain.User{ID: uuid.NewString(), Email: strings.ToLower(email), DisplayName: name, Role: role, Status: "active", CreatedAt: time.Now().UTC()}
	_, err := s.DB.ExecContext(ctx, `INSERT INTO users(id,email,display_name,password_hash,role,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)`, u.ID, u.Email, u.DisplayName, hash, u.Role, u.Status, u.CreatedAt, u.CreatedAt)
	return u, err
}

func (s *Store) QueuedAnalysisIDs(ctx context.Context) ([]string, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id FROM analyses WHERE status IN ('queued','processing') ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			_ = rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	if err = rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err = rows.Close(); err != nil {
		return nil, err
	}
	if _, err = s.DB.ExecContext(ctx, `UPDATE analyses SET status='queued',progress=0 WHERE status='processing'`); err != nil {
		return nil, err
	}
	return ids, nil
}

func (s *Store) UserCredentials(ctx context.Context, email string) (domain.User, string, error) {
	var u domain.User
	var hash string
	err := s.DB.QueryRowContext(ctx, `SELECT id,email,display_name,role,status,created_at,password_hash FROM users WHERE email=?`, strings.ToLower(email)).Scan(&u.ID, &u.Email, &u.DisplayName, &u.Role, &u.Status, &u.CreatedAt, &hash)
	return u, hash, err
}

func (s *Store) User(ctx context.Context, id string) (domain.User, error) {
	var u domain.User
	err := s.DB.QueryRowContext(ctx, `SELECT id,email,display_name,role,status,created_at FROM users WHERE id=?`, id).Scan(&u.ID, &u.Email, &u.DisplayName, &u.Role, &u.Status, &u.CreatedAt)
	return u, err
}

func (s *Store) ListUsers(ctx context.Context) ([]domain.User, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,email,display_name,role,status,created_at FROM users ORDER BY created_at DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := []domain.User{}
	for rows.Next() {
		var u domain.User
		if err = rows.Scan(&u.ID, &u.Email, &u.DisplayName, &u.Role, &u.Status, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (s *Store) UpdateUserStatus(ctx context.Context, id, status string) error {
	if status != "active" && status != "blocked" {
		return fmt.Errorf("invalid user status")
	}
	result, err := s.DB.ExecContext(ctx, `UPDATE users SET status=?,updated_at=? WHERE id=?`, status, time.Now().UTC(), id)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) EnsureAdmin(ctx context.Context, email string) error {
	if strings.TrimSpace(email) == "" {
		return nil
	}
	_, err := s.DB.ExecContext(ctx, `UPDATE users SET role='admin',status='active',updated_at=? WHERE email=?`, time.Now().UTC(), strings.ToLower(strings.TrimSpace(email)))
	return err
}

func (s *Store) CreateAnalysis(ctx context.Context, a domain.Analysis) (domain.Analysis, error) {
	a.ID = uuid.NewString()
	a.Status = "queued"
	a.Progress = 0
	a.CreatedAt = time.Now().UTC()
	_, err := s.DB.ExecContext(ctx, `INSERT INTO analyses(id,user_id,source,external_id,white_name,black_name,result,pgn,status,progress,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, a.ID, a.UserID, a.Source, a.ExternalID, a.WhiteName, a.BlackName, a.Result, a.PGN, a.Status, a.Progress, a.CreatedAt)
	return a, err
}

type scanner interface{ Scan(...any) error }

func scanAnalysis(row scanner) (domain.Analysis, error) {
	var a domain.Analysis
	var uid, external, errText, summary sql.NullString
	var completed sql.NullTime
	err := row.Scan(&a.ID, &uid, &a.Source, &external, &a.WhiteName, &a.BlackName, &a.Result, &a.PGN, &a.Status, &a.Progress, &errText, &summary, &a.CreatedAt, &completed)
	if uid.Valid {
		a.UserID = &uid.String
	}
	if external.Valid {
		a.ExternalID = &external.String
	}
	if errText.Valid {
		a.Error = &errText.String
	}
	if summary.Valid {
		a.Summary = json.RawMessage(summary.String)
	}
	if completed.Valid {
		a.CompletedAt = &completed.Time
	}
	return a, err
}

const analysisColumns = `id,user_id,source,external_id,white_name,black_name,result,pgn,status,progress,error,summary,created_at,completed_at`

func (s *Store) Analysis(ctx context.Context, id string) (domain.Analysis, error) {
	a, err := scanAnalysis(s.DB.QueryRowContext(ctx, `SELECT `+analysisColumns+` FROM analyses WHERE id=?`, id))
	if err != nil {
		return a, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT ply,san,uci,fen_before,fen_after,score_before,score_after,mate_before,mate_after,COALESCE(best_move,''),principal_variation,classification,loss,explanation FROM move_analyses WHERE analysis_id=? ORDER BY ply`, id)
	if err != nil {
		return a, err
	}
	defer rows.Close()
	for rows.Next() {
		var m domain.MoveAnalysis
		var before, after, mateBefore, mateAfter sql.NullInt64
		var pv string
		if err = rows.Scan(&m.Ply, &m.SAN, &m.UCI, &m.FENBefore, &m.FENAfter, &before, &after, &mateBefore, &mateAfter, &m.BestMove, &pv, &m.Classification, &m.Loss, &m.Explanation); err != nil {
			return a, err
		}
		m.ScoreBefore = intPtr(before)
		m.ScoreAfter = intPtr(after)
		m.MateBefore = intPtr(mateBefore)
		m.MateAfter = intPtr(mateAfter)
		_ = json.Unmarshal([]byte(pv), &m.PrincipalVariation)
		a.Moves = append(a.Moves, m)
	}
	return a, rows.Err()
}

func (s *Store) ListAnalyses(ctx context.Context, userID string, admin bool) ([]domain.Analysis, error) {
	query := `SELECT ` + analysisColumns + ` FROM analyses`
	args := []any{}
	if !admin {
		query += ` WHERE user_id=?`
		args = append(args, userID)
	}
	query += ` ORDER BY created_at DESC LIMIT 100`
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.Analysis{}
	for rows.Next() {
		a, scanErr := scanAnalysis(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, a)
	}
	return items, rows.Err()
}

func (s *Store) StartAnalysis(ctx context.Context, id string) (domain.Analysis, error) {
	result, err := s.DB.ExecContext(ctx, `UPDATE analyses SET status='processing',progress=1,error=NULL WHERE id=? AND status IN ('queued','failed')`, id)
	if err != nil {
		return domain.Analysis{}, err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return domain.Analysis{}, errors.New("job is not available")
	}
	return s.Analysis(ctx, id)
}
func (s *Store) SetProgress(ctx context.Context, id string, progress int) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE analyses SET progress=? WHERE id=?`, progress, id)
	return err
}
func (s *Store) CompleteAnalysis(ctx context.Context, id string, moves []domain.MoveAnalysis, summary any) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, m := range moves {
		pv, _ := json.Marshal(m.PrincipalVariation)
		_, err = tx.ExecContext(ctx, `INSERT INTO move_analyses(analysis_id,ply,san,uci,fen_before,fen_after,score_before,score_after,mate_before,mate_after,best_move,principal_variation,classification,loss,explanation) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, id, m.Ply, m.SAN, m.UCI, m.FENBefore, m.FENAfter, m.ScoreBefore, m.ScoreAfter, m.MateBefore, m.MateAfter, m.BestMove, string(pv), m.Classification, m.Loss, m.Explanation)
		if err != nil {
			return err
		}
	}
	body, err := json.Marshal(summary)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE analyses SET status='complete',progress=100,summary=?,completed_at=? WHERE id=?`, string(body), time.Now().UTC(), id)
	if err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Store) FailAnalysis(ctx context.Context, id string, cause error) {
	_, _ = s.DB.ExecContext(ctx, `UPDATE analyses SET status='failed',error=? WHERE id=?`, cause.Error(), id)
}
func IsNotFound(err error) bool { return errors.Is(err, sql.ErrNoRows) }
func intPtr(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	v := int(value.Int64)
	return &v
}

const schema = `
PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;
CREATE TABLE IF NOT EXISTS users(id TEXT PRIMARY KEY,email TEXT NOT NULL UNIQUE,display_name TEXT NOT NULL,password_hash TEXT NOT NULL,role TEXT NOT NULL DEFAULT 'user',status TEXT NOT NULL DEFAULT 'active',created_at DATETIME NOT NULL,updated_at DATETIME NOT NULL);
CREATE TABLE IF NOT EXISTS analyses(id TEXT PRIMARY KEY,user_id TEXT REFERENCES users(id) ON DELETE SET NULL,source TEXT NOT NULL,external_id TEXT,white_name TEXT NOT NULL DEFAULT 'White',black_name TEXT NOT NULL DEFAULT 'Black',result TEXT NOT NULL DEFAULT '*',pgn TEXT NOT NULL,status TEXT NOT NULL DEFAULT 'queued',progress INTEGER NOT NULL DEFAULT 0,error TEXT,summary TEXT,created_at DATETIME NOT NULL,completed_at DATETIME);
CREATE TABLE IF NOT EXISTS move_analyses(id INTEGER PRIMARY KEY AUTOINCREMENT,analysis_id TEXT NOT NULL REFERENCES analyses(id) ON DELETE CASCADE,ply INTEGER NOT NULL,san TEXT NOT NULL,uci TEXT NOT NULL,fen_before TEXT NOT NULL,fen_after TEXT NOT NULL,score_before INTEGER,score_after INTEGER,mate_before INTEGER,mate_after INTEGER,best_move TEXT,principal_variation TEXT NOT NULL DEFAULT '[]',classification TEXT NOT NULL,loss INTEGER NOT NULL DEFAULT 0,explanation TEXT NOT NULL DEFAULT '',UNIQUE(analysis_id,ply));
CREATE INDEX IF NOT EXISTS analyses_user_created_idx ON analyses(user_id,created_at DESC);
CREATE INDEX IF NOT EXISTS analyses_status_idx ON analyses(status,created_at);`
