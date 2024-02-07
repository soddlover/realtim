#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <pthread.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>

#define PORT 20025
#define BUF_SIZE 1024

//10.100.23.129
int send_sockfd;
int rec_sockfd;

void handle_sigint(int sig){
    close(send_sockfd);
    close(rec_sockfd);
    printf("Ports closed motherfucker");
    exit(0);
}

void* send_udp(void* arg) {
    
    struct sockaddr_in servaddr;

    // Creating socket file descriptor
    if ((send_sockfd = socket(AF_INET, SOCK_DGRAM, 0)) < 0) {
        perror("socket creation failed");
        exit(EXIT_FAILURE);
    }

    memset(&servaddr, 0, sizeof(servaddr));
    //inet_pton(AF_INET,"0.0.0.255", &(servaddr.sin_addr));


    // Filling server information
    servaddr.sin_family = AF_INET;
    servaddr.sin_addr.s_addr = INADDR_ANY;
    servaddr.sin_port = htons(PORT);


    int n;
    char buffer[BUF_SIZE] = "Velkommen etter n00bs, get rekt";
    while(1){
        sendto(send_sockfd, (const char *)buffer, strlen(buffer),
            MSG_CONFIRM, (const struct sockaddr *) &servaddr, 
                sizeof(servaddr));
        printf("Message sent.\n");
        sleep(1);
    }
    close(send_sockfd);
    return NULL;
}

void* receive_udp(void* arg) {
    char buffer[BUF_SIZE];
    struct sockaddr_in servaddr, cliaddr;

    // Creating socket file descriptor
    if ((rec_sockfd = socket(AF_INET, SOCK_DGRAM, 0)) < 0) {
        perror("socket creation failed");
        exit(EXIT_FAILURE);
    }
    
    memset(&servaddr, 0, sizeof(servaddr));
    memset(&cliaddr, 0, sizeof(cliaddr));
    //inet_pton(AF_INET,"255.255.255.255", &(servaddr.sin_addr));

    // Filling server information
    servaddr.sin_family = AF_INET; // IPv4
    servaddr.sin_addr.s_addr = INADDR_ANY;
    servaddr.sin_port = htons(PORT);

    // Bind the socket with the server address
    if (bind(rec_sockfd, (const struct sockaddr *)&servaddr, 
            sizeof(servaddr)) < 0)
    {
        perror("bind failed");
        exit(EXIT_FAILURE);
    }

    int len, n;
    len = sizeof(cliaddr); //len is value/resuslt

    while(1){
    n = recvfrom(rec_sockfd, (char *)buffer, BUF_SIZE, 
                MSG_WAITALL, (struct sockaddr *) &cliaddr,
                &len);
    buffer[n] = '\0';
    printf("Client : %s\n", buffer);
    }
    close(rec_sockfd);
    return NULL;
}

int main() {
    pthread_t send_thread, receive_thread;
    pthread_create(&receive_thread, NULL, receive_udp, NULL);
    sleep(1);
    pthread_create(&send_thread, NULL, send_udp, NULL);
    

    pthread_join(send_thread, NULL);
    pthread_join(receive_thread, NULL);
    return 0;
}