#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <pthread.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>

#define PORT 30000
#define BUF_SIZE 1024

//10.100.23.129

void* send_udp(void* arg) {
    int sockfd;
    struct sockaddr_in servaddr;

    // Creating socket file descriptor
    if ((sockfd = socket(AF_INET, SOCK_DGRAM, 0)) < 0) {
        perror("socket creation failed");
        exit(EXIT_FAILURE);
    }

    memset(&servaddr, 0, sizeof(servaddr));

    // Filling server information
    servaddr.sin_family = AF_INET;
    servaddr.sin_port = htons(PORT);
    servaddr.sin_addr.s_addr = INADDR_ANY;

    int n;
    char buffer[BUF_SIZE] = "Balls";
    while (1){
        sendto(sockfd, (const char *)buffer, strlen(buffer),
            MSG_CONFIRM, (const struct sockaddr *) &servaddr, 
                sizeof(servaddr));
        printf("Message sent.\n");
        sleep(5);
    }
    close(sockfd);
    return NULL;
}

void* receive_udp(void* arg) {
    int sockfd;
    char buffer[BUF_SIZE];
    struct sockaddr_in servaddr, cliaddr;

    // Creating socket file descriptor
    if ((sockfd = socket(AF_INET, SOCK_DGRAM, 0)) < 0) {
        perror("socket creation failed");
        exit(EXIT_FAILURE);
    }

    memset(&servaddr, 0, sizeof(servaddr));
    memset(&cliaddr, 0, sizeof(cliaddr));

    // Filling server information
    servaddr.sin_family = AF_INET; // IPv4
    servaddr.sin_addr.s_addr = INADDR_ANY;
    servaddr.sin_port = htons(PORT);

    // Bind the socket with the server address
    if (bind(sockfd, (const struct sockaddr *)&servaddr, 
            sizeof(servaddr)) < 0)
    {
        perror("bind failed");
        exit(EXIT_FAILURE);
    }

    int len, n;
    len = sizeof(cliaddr); //len is value/resuslt

    while (1){
        n = recvfrom(sockfd, (char *)buffer, BUF_SIZE, 
                    MSG_WAITALL, (struct sockaddr *) &cliaddr,
                    &len);
        buffer[n] = '\0';
        printf("Client : %s\n", buffer);
    }
    close(sockfd);
    return NULL;
}

int main() {
    pthread_t send_thread, receive_thread;

    pthread_create(&send_thread, NULL, send_udp, NULL);
    pthread_create(&receive_thread, NULL, receive_udp, NULL);

    pthread_join(send_thread, NULL);
    pthread_join(receive_thread, NULL);

    return 0;
}