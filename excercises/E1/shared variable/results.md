

The magic number prints as all kinds of silly values each time the program is run. This is because we share a global varibale and we do read and write operations on it. Data race!

When setting runtime.GOMAXPROCS(1) we only allow the program to use 1 thread. Therefore program works as it should and prints macig number 0. This is because each for loop is completed before the next is run. 