CREATE TABLE powerccdr (   id MEDIUMINT NOT NULL AUTO_INCREMENT,   tm TIMESTAMP NOT NULL,   duration int(11) DEFAULT 0,   called varchar(255) DEFAULT NULL,   calling varchar(255) DEFAULT NULL,   PRIMARY KEY (id) );
CREATE TABLE smsto (   id MEDIUMINT NOT NULL AUTO_INCREMENT,   phone varchar(255) DEFAULT NULL UNIQUE,   name varchar(255) DEFAULT NULL, sendsms TINYINT NOT NULL DEFAULT 0, avayacp TINYINT DEFAULT 0,  PRIMARY KEY (id) ) CHARACTER SET=utf8;