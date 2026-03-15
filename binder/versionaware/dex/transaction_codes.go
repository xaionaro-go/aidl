package dex

// TransactionCodes maps AIDL method names to their binder transaction codes.
// Method names use the original AIDL camelCase (e.g., "isUserAMonkey").
type TransactionCodes map[string]uint32
