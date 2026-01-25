package database

func Migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		display_name TEXT NOT NULL,
		description TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS books (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		google_books_id TEXT NOT NULL UNIQUE,
		title TEXT NOT NULL,
		authors TEXT DEFAULT '',
		thumbnail_url TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS user_books (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		book_id INTEGER NOT NULL,
		shelf TEXT NOT NULL CHECK(shelf IN ('want_to_read', 'currently_reading', 'read')),
		sub_status TEXT DEFAULT NULL,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (book_id) REFERENCES books(id) ON DELETE CASCADE,
		UNIQUE(user_id, book_id)
	);

	CREATE TABLE IF NOT EXISTS sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		token TEXT NOT NULL UNIQUE,
		expires_at DATETIME NOT NULL
	);
	`
	_, err := DB.Exec(schema)
	if err != nil {
		return err
	}

	// Run incremental migrations
	return runMigrations()
}

func runMigrations() error {
	migrations := []struct {
		name string
		sql  string
	}{
		{
			name: "add_password_hash_to_users",
			sql:  "ALTER TABLE users ADD COLUMN password_hash TEXT DEFAULT NULL",
		},
		{
			name: "add_user_id_to_sessions",
			sql:  "ALTER TABLE sessions ADD COLUMN user_id INTEGER DEFAULT NULL REFERENCES users(id) ON DELETE CASCADE",
		},
		{
			name: "add_session_type_to_sessions",
			sql:  "ALTER TABLE sessions ADD COLUMN session_type TEXT DEFAULT 'admin' NOT NULL",
		},
		{
			name: "add_added_at_to_user_books",
			sql:  "ALTER TABLE user_books ADD COLUMN added_at DATETIME DEFAULT NULL",
		},
		{
			name: "add_started_reading_at_to_user_books",
			sql:  "ALTER TABLE user_books ADD COLUMN started_reading_at DATETIME DEFAULT NULL",
		},
		{
			name: "add_finished_reading_at_to_user_books",
			sql:  "ALTER TABLE user_books ADD COLUMN finished_reading_at DATETIME DEFAULT NULL",
		},
		{
			name: "add_profile_picture_to_users",
			sql:  "ALTER TABLE users ADD COLUMN profile_picture TEXT DEFAULT NULL",
		},
		{
			name: "add_theme_to_users",
			sql:  "ALTER TABLE users ADD COLUMN theme TEXT DEFAULT 'light'",
		},
		{
			name: "add_isbn_13_to_books",
			sql:  "ALTER TABLE books ADD COLUMN isbn_13 TEXT DEFAULT NULL",
		},
		{
			name: "add_isbn_10_to_books",
			sql:  "ALTER TABLE books ADD COLUMN isbn_10 TEXT DEFAULT NULL",
		},
		{
			name: "add_page_count_to_books",
			sql:  "ALTER TABLE books ADD COLUMN page_count INTEGER DEFAULT NULL",
		},
		{
			name: "create_isbn_13_index",
			sql:  "CREATE INDEX IF NOT EXISTS idx_books_isbn_13 ON books(isbn_13)",
		},
		{
			name: "create_isbn_10_index",
			sql:  "CREATE INDEX IF NOT EXISTS idx_books_isbn_10 ON books(isbn_10)",
		},
		{
			name: "create_events_table",
			sql: `CREATE TABLE IF NOT EXISTS events (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				user_id INTEGER NOT NULL,
				event_type TEXT NOT NULL,
				book_id INTEGER,
				shelf TEXT,
				old_value TEXT,
				new_value TEXT,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
				FOREIGN KEY (book_id) REFERENCES books(id) ON DELETE CASCADE
			)`,
		},
		{
			name: "create_events_user_id_index",
			sql:  "CREATE INDEX IF NOT EXISTS idx_events_user_id ON events(user_id)",
		},
		{
			name: "create_events_created_at_index",
			sql:  "CREATE INDEX IF NOT EXISTS idx_events_created_at ON events(created_at DESC)",
		},
		{
			name: "add_rating_to_user_books",
			sql:  "ALTER TABLE user_books ADD COLUMN rating INTEGER DEFAULT NULL CHECK(rating IS NULL OR (rating >= 1 AND rating <= 5))",
		},
		{
			name: "add_description_to_books",
			sql:  "ALTER TABLE books ADD COLUMN description TEXT DEFAULT NULL",
		},
	}

	// Create migrations table if not exists
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name TEXT PRIMARY KEY,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	for _, m := range migrations {
		// Check if migration already applied
		var count int
		err := DB.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE name = ?", m.name).Scan(&count)
		if err != nil {
			return err
		}
		if count > 0 {
			continue
		}

		// Apply migration
		_, err = DB.Exec(m.sql)
		if err != nil {
			// Ignore "duplicate column" errors for idempotent migrations
			continue
		}

		// Record migration
		_, err = DB.Exec("INSERT INTO schema_migrations (name) VALUES (?)", m.name)
		if err != nil {
			return err
		}
	}

	return nil
}
