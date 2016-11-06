CREATE TABLE `api_server_group` (
    `id` bigint(20) NOT NULL PRIMARY KEY AUTOINCREMENT,
    `status` bigint(20) NOT NULL DEFAULT 1 ,
    `name` varchar(64) NOT NULL DEFAULT '' ,
    `intro` varchar(10000) NOT NULL DEFAULT '' ,
    `home_page` varchar(1024) NOT NULL DEFAULT '' ,
    `ctime` datetime NOT NULL,
    `mtime` datetime NOT NULL
);
CREATE TABLE `api_server` (
    `id` bigint(20) NOT NULL PRIMARY KEY AUTOINCREMENT,
    `status` bigint(20) NOT NULL DEFAULT 0 ,
    `group_id` bigint(20) NOT NULL DEFAULT 0 ,
    `name` varchar(64) NOT NULL DEFAULT '' ,
    `intro` varchar(10000) NOT NULL DEFAULT '' ,
    `port` bigint(20) NOT NULL DEFAULT 0 ,
    `uniq_key` varchar(32) NOT NULL DEFAULT ''  UNIQUE,
    `ctime` datetime NOT NULL,
    `mtime` datetime NOT NULL
);
CREATE INDEX `api_server_group_id` ON `api_server` (`group_id`);
CREATE TABLE `api_item` (
    `id` bigint(20) NOT NULL PRIMARY KEY AUTOINCREMENT,
    `status` bigint(20) NOT NULL DEFAULT 0 ,
    `name` varchar(64) NOT NULL DEFAULT '' ,
    `intro` varchar(10000) NOT NULL DEFAULT '' ,
    `location` varchar(255) NOT NULL DEFAULT '' ,
    `ctime` datetime NOT NULL,
    `mtime` datetime NOT NULL,
    `server_id` bigint(20) NOT NULL DEFAULT 0 
);
CREATE INDEX `api_item_server_id` ON `api_item` (`server_id`);
CREATE TABLE `api_item_host` (
    `id` bigint(20) NOT NULL PRIMARY KEY AUTOINCREMENT,
    `status` bigint(20) NOT NULL DEFAULT 0 ,
    `name` varchar(64) NOT NULL DEFAULT '' ,
    `intro` varchar(255) NOT NULL DEFAULT '' ,
    `url` varchar(10000) NOT NULL DEFAULT '' ,
    `ctime` datetime NOT NULL,
    `mtime` datetime NOT NULL,
    `item_id` bigint(20) NOT NULL DEFAULT 0 
);
CREATE INDEX `api_item_host_item_id` ON `api_item_host` (`item_id`);