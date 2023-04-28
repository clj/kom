.print "Running test script:"
.bail on
.echo on
.load kom
CREATE TABLE inventree_settings (key, value);
INSERT INTO inventree_settings VALUES ("server", "http://localhost:45454");
INSERT INTO inventree_settings VALUES ("username", "username");
INSERT INTO inventree_settings VALUES ("password", "password");
CREATE VIRTUAL TABLE test USING kom(plugin=inventree, settings=inventree_settings);
.schema test
SELECT * FROM test;
SELECT * FROM inventree_settings;
