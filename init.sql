create table IF NOT EXISTS "user" ("id" text not null primary key, "name" text not null, "email" text not null unique, "emailVerified" boolean not null, "image" text, "createdAt" timestamp not null, "updatedAt" timestamp not null, "username" text unique, "isAnonymous" boolean);

create table IF NOT EXISTS "session" ("id" text not null primary key, "expiresAt" timestamp not null, "token" text not null unique, "createdAt" timestamp not null, "updatedAt" timestamp not null, "ipAddress" text, "userAgent" text, "userId" text not null references "user" ("id"));

create table IF NOT EXISTS "account" ("id" text not null primary key, "accountId" text not null, "providerId" text not null, "userId" text not null references "user" ("id"), "accessToken" text, "refreshToken" text, "idToken" text, "accessTokenExpiresAt" timestamp, "refreshTokenExpiresAt" timestamp, "scope" text, "password" text, "createdAt" timestamp not null, "updatedAt" timestamp not null);

create table IF NOT EXISTS "verification" ("id" text not null primary key, "identifier" text not null, "value" text not null, "expiresAt" timestamp not null, "createdAt" timestamp, "updatedAt" timestamp);


CREATE TABLE IF NOT EXISTS customers (
	userid TEXT REFERENCES "user"(id) ON UPDATE CASCADE,
	customer_id INT,
	order_id INT, 
	order_item_id INT, 
	product_id INT, 
	variant_id INT, 
	user_name TEXT, 
	user_email TEXT, 
	status TEXT, 
	status_formatted TEXT,
	PRIMARY KEY(userid, customer_id)
);

CREATE TABLE IF NOT EXISTS leaderboards (
	id INT GENERATED ALWAYS AS IDENTITY UNIQUE,
	created_by TEXT REFERENCES "user"(id) ON UPDATE CASCADE,
	display_name TEXT,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP, 
	highest_first BOOLEAN NOT NULL DEFAULT TRUE,
	is_time BOOLEAN NOT NULL DEFAULT FALSE,
	uses_verification BOOLEAN NOT NULL DEFAULT FALSE,
	duration INTERVAL,
	start TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	multiple_submissions BOOLEAN DEFAULT TRUE,
	PRIMARY KEY(id, created_by)
);



CREATE TABLE IF NOT EXISTS submissions (
	id INT GENERATED ALWAYS AS IDENTITY UNIQUE,
	leaderboard INT REFERENCES leaderboards(id),
	userid TEXT REFERENCES "user"(id) ON UPDATE CASCADE,
	link TEXT,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	score NUMERIC NOT NULL,
	verified BOOLEAN NOT NULL DEFAULT FALSE,
	last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP, 
	PRIMARY KEY(id, leaderboard, userid)
);


CREATE TABLE IF NOT EXISTS verifiers (
	leaderboard INT REFERENCES leaderboards(id),
	userid TEXT REFERENCES "user"(id) ON UPDATE CASCADE,
	added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY(leaderboard, userid)
);


CREATE OR REPLACE FUNCTION function_update_timestamp() RETURNS TRIGGER AS
$BODY$
BEGIN
	UPDATE leaderboards SET last_updated=NOW() WHERE NEW.leaderboard=leaderboards.id;
        RETURN NEW;
END;
$BODY$
language plpgsql;


CREATE TRIGGER trig_update_time
     AFTER INSERT OR UPDATE ON submissions
     FOR EACH ROW
     EXECUTE FUNCTION function_update_timestamp();

