CREATE TABLE IF NOT EXISTS powerccdr (  id MEDIUMINT NOT NULL AUTO_INCREMENT,
                                        tm TIMESTAMP NOT NULL,
                                        originaldt varchar(255) DEFAULT NULL,
                                        duration int(11) DEFAULT 0,
                                        called varchar(255) DEFAULT NULL,
                                        calling varchar(255) DEFAULT NULL,
                                        cond CHAR(1) NOT NULL DEFAULT '0',
                                        PRIMARY KEY (id),
                                        INDEX idx_tm (tm),
                                        INDEX idx_called (called),
                                        INDEX idx_calling (calling)
                                        )CHARACTER SET=utf8;

CREATE TABLE IF NOT EXISTS smsto (   id MEDIUMINT NOT NULL AUTO_INCREMENT,
                                     phone varchar(255) DEFAULT NULL UNIQUE,
                                     name varchar(255) DEFAULT NULL,
                                     sendsms TINYINT NOT NULL DEFAULT 0,
                                     avayacp TINYINT DEFAULT 0,
                                     PRIMARY KEY (id) ) CHARACTER SET=utf8;