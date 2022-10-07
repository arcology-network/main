#pragma once

#include <stdlib.h>
#include <stdint.h>
#include <stdbool.h>

#ifdef __cplusplus
extern "C" {
#endif
	void* Start(char* fileName, uint32_t length, char* msgBuffer);
	void Stop(void* enginePtr);
	void SaveHistory(void* enginePtr, char* msgBuffer);

	void Schedule(void* enginePtr, char* calleePtr, uint32_t* txIdPtr, uint32_t* newTxOrder, uint32_t* branches, uint32_t* generations, uint32_t count, char* msgBuffer);
	void UpdateHistory(void* enginePtr, char* lft, char* rgt, char* where, uint32_t count, char* msgBuffer);

	void GetVersion(char* version); // Get version 
	void GetProduct(char* product); // Get product name

#ifdef __cplusplus
}
#endif
