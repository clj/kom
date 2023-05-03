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
CREATE VIRTUAL TABLE Capacitors USING kom(plugin=inventree, settings=settings, categories="Capacitors", default_symbol="Device:C", fields="Category:(int)category, Active:(int)active=(int)0, FullName:full_name");
.schema Passives
SELECT * FROM Passives;
SELECT * FROM Passives LIMIT 1;
SELECT * FROM Passives WHERE PK=30;
SELECT * FROM Capacitors;
SELECT * FROM settings;
select kom_version();
select kom_version("version");
select kom_version("sha");
select kom_version("build_date");
