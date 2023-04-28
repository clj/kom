.print "Running test script:"
.bail on
.echo on
.load kom
CREATE VIRTUAL TABLE test USING kom(plugin=inventree);
.schema test
SELECT * FROM test;
