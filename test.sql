.print "Running test script:"
.bail on
.echo on
.load ./kom
CREATE TABLE settings (key, value);
INSERT INTO settings VALUES ("server", "http://localhost:45454");
INSERT INTO settings VALUES ("username", "username");
INSERT INTO settings VALUES ("password", "password");
CREATE VIRTUAL TABLE Passives USING kom(plugin=inventree, settings=settings, categories="Capacitors,Resistors", default_symbol="Device:R", default_footprint="SMD:0506");
CREATE VIRTUAL TABLE Resistors USING kom(plugin=inventree, settings=settings, categories="Resistors", default_symbol="Device:R", default_footprint="Resistor_SMD:R_0805_2012Metric");
.schema Passives
SELECT * FROM Passives;
SELECT * FROM Passives LIMIT 1;
SELECT * FROM Passives WHERE PK=30;
SELECT * FROM settings;
