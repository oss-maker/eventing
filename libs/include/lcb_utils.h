// Copyright (c) 2019 Couchbase, Inc.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an "AS IS"
// BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
// or implied. See the License for the specific language governing
// permissions and limitations under the License.

#ifndef COUCHBASE_LCB_UTILS_H
#define COUCHBASE_LCB_UTILS_H

#include "log.h"
#include <libcouchbase/couchbase.h>

struct Result {
  lcb_CAS cas;
  lcb_error_t rc;
  std::string value;
  uint32_t exptime;
  int64_t counter;
  Result() : cas(0), rc(LCB_SUCCESS) {}
};

const char *GetUsername(void *cookie, const char *host, const char *port,
                        const char *bucket);

const char *GetPassword(void *cookie, const char *host, const char *port,
                        const char *bucket);

// lcb related callbacks
void get_callback(lcb_t instance, int, const lcb_RESPBASE *rb);

void set_callback(lcb_t instance, int cbtype, const lcb_RESPBASE *rb);

void sdmutate_callback(lcb_t instance, int cbtype, const lcb_RESPBASE *rb);

void del_callback(lcb_t instance, int cbtype, const lcb_RESPBASE *rb);

void counter_callback(lcb_t instance, int cbtype, const lcb_RESPBASE *rb);

extern struct lcb_logprocs_st evt_logger;
#endif // COUCHBASE_LCB_UTILS_H