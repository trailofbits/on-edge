//====================================================================================================//
// Copyright 2019 Trail of Bits
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//====================================================================================================//

// This file provides access to two symbols within ThreadSanitizer, which is part of the Go runtime.
// The symbols are given "weak" implementations so that they should be overridden be the real ("strong")
// implementations at linktime.  Moreover, the weak implementations simply call __builtin_trap() so that
// if something were to go wrong with the linking and they were to be called at runtime, then they will
// cause the program to abort.

struct SuppressionContext;

//====================================================================================================//

// __tsan::Suppressions()
struct SuppressionContext *_ZN6__tsan12SuppressionsEv() __attribute__((weak));
struct SuppressionContext *_ZN6__tsan12SuppressionsEv() {
  __builtin_trap();
}
struct SuppressionContext *__tsan_Suppressions() {
  return _ZN6__tsan12SuppressionsEv();
}

//====================================================================================================//

// __sanitizer::SuppressionContext::Parse(char const*)
int _ZN11__sanitizer18SuppressionContext5ParseEPKc(struct SuppressionContext *this, const char *value)
  __attribute__((weak));
int _ZN11__sanitizer18SuppressionContext5ParseEPKc(struct SuppressionContext *this, const char *value) {
  __builtin_trap();
}
int __sanitizer_SuppressionContext_Parse(struct SuppressionContext *this, const char *value) {
  return _ZN11__sanitizer18SuppressionContext5ParseEPKc(this, value);
}

//====================================================================================================//
