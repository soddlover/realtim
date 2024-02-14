// Compile with `gcc foo.c -Wall -std=gnu99 -lpthread`, or use the makefile
// The executable will be named `foo` if you use the makefile, or `a.out` if you use gcc directly

#include <pthread.h>
#include <stdio.h>

int i = 0;
pthread_mutex_t lock;

//We use mutex since this is suted for a binary resource which can only be unlocked by the same process that locked it.
//Semaphore is for signalling and deciding how many proccesses can use a resource (often more than 1). Everyone has the "key"


// Note the return type: void*
void* incrementingThreadFunction(){
    // TODO: increment i 1_000_000 times
    for (int j=1;j<1000000; j++){
        pthread_mutex_lock(&lock);
        i=i+1;
        pthread_mutex_unlock(&lock);
    }

    return NULL;
}

void* decrementingThreadFunction(){
    // TODO: decrement i 1_000_000 times
    for (int j=1;j<1000000; j++){
        pthread_mutex_lock(&lock);
        i=i-1;
        pthread_mutex_unlock(&lock);
    }
    return NULL;
}


int main(){
    // TODO: 
    // start the two functions as their own threads using `pthread_create`
    // Hint: search the web! Maybe try "pthread_create example"?
    pthread_t thread_id_1;
    pthread_t thread_id_2;

    pthread_mutex_init(&lock, NULL);

    pthread_create(&thread_id_1, NULL, incrementingThreadFunction, NULL);
    pthread_create(&thread_id_2, NULL, decrementingThreadFunction, NULL);
    // TODO:
    // wait for the two threads to be done before printing the final result
    // Hint: Use `pthread_join`    
    pthread_join(thread_id_1,NULL);
    pthread_join(thread_id_2,NULL);


    printf("The magic number is: %d\n", i);
    return 0;
}
