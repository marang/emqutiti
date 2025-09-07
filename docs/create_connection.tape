Set Width 1600
Set Height 1000
Set FontSize 18
Set Padding 16
Type "./emqutiti"
Enter
Sleep 1200ms
# Open Add broker form
Type "a"
Sleep 400ms
# Name
Type "local"
Sleep 200ms
# Schema (leave default tcp)
Tab
Sleep 100ms
# Host
Tab
Type "broker.hivemq.com"
Sleep 150ms
# Port
Tab
Type "1883"
Sleep 150ms
# Toggle "Load from env" to avoid keyring writes
Tab
Tab
Tab
Tab
Tab
Space
Sleep 3000ms
# Save
Enter
Sleep 4000ms
Ctrl+D
