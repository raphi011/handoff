CREATE TABLE TestSuiteRun(
    id INTEGER PRIMARY KEY,
    suiteName TEXT NOT NULL,
    environment TEXT NOT NULL,
    result TEXT,
    testFilter TEXT,

    total INTEGER NOT NULL,
    passed INTEGER NOT NULL,
    skipped INTEGER NOT NULL,
    failed INTEGER NOT NULL,

    scheduledTime TEXT NOT NULL,
    startTime TEXT,
    endTime TEXT, 

    setupLogs TEXT,

    triggeredBy TEXT NOT NULL

) STRICT;

CREATE TABLE TestRun(
    suiteRunId INTEGER NOT NULL,
    testName TEXT NOT NULL,
    passed INTEGER NOT NULL, 
    skipped INTEGER NOT NULL,

    logs TEXT,

    startTime TEXT,
    endTime TEXT,

    PRIMARY KEY(suiteRunId, testName),
    FOREIGN KEY(suiteRunId) REFERENCES TestSuiteRun(id)

) STRICT;
