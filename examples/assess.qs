# assess.qs — a runnable example of the shared QYVORA scripting engine driving
# sekhmet's fuzzing assessment.
#
#     sekhmet script run examples/assess.qs
#
# sekhmet runs against the deterministic simulation target here: no real
# binary, no authorization. Live fuzzing requires an authorized target
# (sekhmet target set --path <binary> --authorize).

# Use the deterministic simulation target (no real binary).
set assess.sim -> "true"

# Run the assessment: set the simulation target and a bounded fuzzing campaign
# (one worker, a small execution cap). A specific capability id such as
# "sekhmet.campaign.fuzz" runs just that operation.
run assess

# Read live properties from the completed campaign via $(...).
echo -> "session    = $(session.id)"
echo -> "executions = $(result.executions)"
echo -> "corpus     = $(result.corpus)"
echo -> "behaviors  = $(result.behaviors)"
echo -> "crashes    = $(result.crashes)"
echo -> "duration   = $(result.duration)"

# List findings and write a JSON campaign report.
run sekhmet.findings.list
report -> "sekhmet-report.json"

echo -> "done"
