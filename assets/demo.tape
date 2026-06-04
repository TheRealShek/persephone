Output demo.gif
Set Theme "Catppuccin Mocha"
Set FontSize 16
Set Width 750
Set Height 450
Set Padding 30
Set TypingSpeed 50ms

Type "mkdir demo-repo && cd demo-repo"
Enter
Enter
Sleep 500ms

Type "purr init"
Enter
Sleep 800ms
Ctrl+L

Type "echo 'hello purr' > README.md && touch main.go"
Enter
Enter
Sleep 500ms

Type "purr add ."
Enter
Sleep 800ms
Ctrl+L

Type "purr ls"
Enter
Enter
Sleep 1s

Type 'purr commit -m "initial commit"'
Enter
Sleep 1s
Ctrl+L

Type "purr log"
Enter
Sleep 2s

Hide
Type "cd .. && rm -rf demo-repo"
Enter
