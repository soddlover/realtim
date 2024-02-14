#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <pthread.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <unistd.h>
#include <signal.h>

#define PORT 33546
#define BUF_SIZE 1024

int sockfd;

void handle_sigint(int sig){
    close(sockfd);
    printf("Socket closed.\n");
    exit(0);
}

void* send_receive_tcp(void* arg) {
    struct sockaddr_in servaddr;

    // Creating socket file descriptor
    if ((sockfd = socket(AF_INET, SOCK_STREAM, 0)) < 0) {
        perror("socket creation failed");
        exit(EXIT_FAILURE);
    }

    memset(&servaddr, 0, sizeof(servaddr));
    //10.100.23.129
    inet_pton(AF_INET,"10.100.23.129", &(servaddr.sin_addr));
    // Filling server information
    servaddr.sin_family = AF_INET;
    //servaddr.sin_addr.s_addr = INADDR_ANY;
    servaddr.sin_port = htons(PORT);

    if (connect(sockfd, (struct sockaddr*)&servaddr, sizeof(servaddr)) < 0) {
        perror("connect failed");
        exit(EXIT_FAILURE);
    }

    char buffer[BUF_SIZE] = "Connect to: 10.100.23.34:33546\0";
    while(1){
        send(sockfd, buffer, strlen(buffer), 0);
        printf("Message sent.\n");
        memset(buffer,0,BUF_SIZE);
        int n = recv(sockfd, buffer, BUF_SIZE, 0);
        if (n <= 0) {
            break; // Connection closed or error
        }
        buffer[n] = '\0';
        printf("Server : %s\n", buffer);

        sleep(1);
        strcpy(buffer, "jalla\0");
    }
    close(sockfd);
    return NULL;
}

int main() {
    pthread_t send_receive_thread;
    signal(SIGINT, handle_sigint);

    pthread_create(&send_receive_thread, NULL, send_receive_tcp, NULL);
    pthread_join(send_receive_thread, NULL);

    return 0;
}