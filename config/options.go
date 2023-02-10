package config

// choice of algorithm to compute layers
const LayerAlgorithm = 0

// Model 1: All servers have an independent probability f of being adversarial
// Model 2: At most n * f servers are adversarial
const Model = 2

// probability 2^-64 of an all adversarial anytrust group
const AnytrustGroupSecurityFactor = -64

// total variational distance less than 2^-64
const ShuffleSecurityFactor = -64

// probability 2^-32 of a link overflow
// const LinkOverflowProbability = -32

// Stream packet size = 2MB
const StreamSize = 2 * 1024 * 1024

// Batch verification of signatures
const BatchSize = 64

// To estimate timeouts
const Bandwidth = 1000 // mega bits per second
// the minimum amount of bytes per read system call
const TCPReadSize = 1460

const MASTER_GROUP = 0

// INSECURE: just for computing messages faster to test other parts of the system
const SkipToken = true

const LogTimes = false

const PreExpandKeys = false

const NoDummies = true
