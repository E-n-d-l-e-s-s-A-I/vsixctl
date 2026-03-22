Output demo.gif

Require vsixctl

Set Shell "bash"
Set FontSize 18
Set Width 1300
Set Height 560
Set TypingSpeed 50ms
Set WindowBar Colorful

Sleep 500ms

# Поиск расширений
Type "vsixctl search python -n 3"
Sleep 500ms
Enter
Wait+Screen@10s /3\./
Sleep 1s

# Установка расширения
Type "vsixctl install ms-python.python"
Sleep 500ms
Enter
Wait+Screen@10s /Proceed/
Sleep 1500ms
Type "y"
Enter
Sleep 5s
