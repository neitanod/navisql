# Define colors
if [ -t 1 ]; then # If stdout connected to terminal
    ESC=$'\033';
    BLACK=$ESC[30m; RED=$ESC[31m; GREEN=$ESC[32m; YELLOW=$ESC[33m; BLUE=$ESC[34m;
    MAGENTA=$ESC[35m; CYAN=$ESC[36m; WHITE=$ESC[37m; GREY=$ESC[90m; NORMAL=$ESC[39m;
    BLACK_BG=$ESC[40m; RED_BG=$ESC[41m; GREEN_BG=$ESC[42m; YELLOW_BG=$ESC[43m; BLUE_BG=$ESC[44m;
    MAGENTA_BG=$ESC[45m; CYAN_BG=$ESC[46m; WHITE_BG=$ESC[47m; GREY_BG=$ESC[100m; NORMAL_BG=$ESC[49m;
fi
