set encoding utf8

################################################################################

set terminal pngcairo size 4096, 2048 lw 2

set grid
set key above
set xlabel 'ms'
set style data lines
set autoscale xfix
set lmargin at screen 0.05
set datafile separator ','
set output 'telemetry.png'

set multiplot layout 7,1 rowsfirs

plot 'telemetry.csv' using 2:11 title 'throttle (us)'
plot 'telemetry.csv' using 2:10 title 'r/min'
plot 'telemetry.csv' using 2:7 title 'I (A)'
plot 'telemetry.csv' using 2:8 title 'U (V)'
plot 'telemetry.csv' using 2:9 title 'P (W)'
plot 'telemetry.csv' using 2:3 title 'loadcell'
plot 'telemetry.csv' using 2:12 title 'gyroX', '' using 2:13 title 'gyroY', '' using 2:14 title 'gyroZ'

unset multiplot
