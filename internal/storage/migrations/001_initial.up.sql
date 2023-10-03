CREATE TABLE  TestSuiteMeta(
    suiteName TEXT PRIMARY KEY, 
    flaky INTEGER NOT NULL
) STRICT;

CREATE TABLE TestSuiteRun(
    id INTEGER NOT NULL,
    suiteName TEXT NOT NULL,

    environment TEXT NOT NULL,
    result TEXT NOT NULL,
    flaky INTEGER NOT NULL,
    scheduledTime TEXT NOT NULL,
    startTime TEXT NOT NULL,
    endTime TEXT NOT NULL, 
    setupLogs TEXT NOT NULL,

    -- options
    triggeredBy TEXT NOT NULL,
    reference TEXT NOT NULL,
    maxTestAttempts INT NOT NULL,
    testFilter TEXT NOT NULL,
    timeoutDuration TEXT NOT NULL,

    PRIMARY KEY(id, suiteName)
) STRICT;

CREATE INDEX TsrName ON TestSuiteRun(suiteName);
CREATE INDEX TsrReference ON TestSuiteRun(reference);

CREATE TABLE TestRun(
    suiteName TEXT NOT NULL,
    suiteRunId INTEGER NOT NULL,
    testName TEXT NOT NULL,
    attempt INTEGER NOT NULL,

    context TEXT NOT NULL,
    result TEXT NOT NULL, 
    compressedLogs BLOB,
    softFailure INTEGER NOT NULL,
    startTime TEXT NOT NULL,
    endTime TEXT NOT NULL,

    PRIMARY KEY(suiteName, suiteRunId, testName, attempt),
    FOREIGN KEY(suiteName, suiteRunId) REFERENCES TestSuiteRun(suiteName, id)
) STRICT;
