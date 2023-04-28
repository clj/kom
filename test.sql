.print "Running test script:"
.echo on
.load kom
CREATE VIRTUAL TABLE test USING kom();
.schema test
SELECT * FROM test;
