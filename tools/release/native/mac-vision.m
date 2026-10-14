#import <Foundation/Foundation.h>
#import <Vision/Vision.h>

// Standalone stdin JSON helper; shipped with each macOS package. No network,
// API key or developer toolchain is needed on the learner's machine.
int main(void) {
    @autoreleasepool {
        NSData *input = [[NSFileHandle fileHandleWithStandardInput] readDataToEndOfFile];
        NSDictionary *request = [NSJSONSerialization JSONObjectWithData:input options:0 error:nil];
        if (![request isKindOfClass:[NSDictionary class]]) return 2;
        VNRecognizeTextRequest *recognize = [[VNRecognizeTextRequest alloc] init];
        recognize.recognitionLevel = VNRequestTextRecognitionLevelAccurate;
        if (@available(macOS 13.0, *)) recognize.revision = VNRecognizeTextRequestRevision3;
        NSError *error = nil;
        NSArray *languages = [VNRecognizeTextRequest supportedRecognitionLanguagesForTextRecognitionLevel:recognize.recognitionLevel revision:recognize.revision error:&error];
        BOOL available = error == nil && [languages containsObject:@"zh-Hans"];
        NSMutableDictionary *result = [@{@"available": @(available)} mutableCopy];
        if ([request[@"mode"] isEqual:@"recognize"] && available) {
            if (![request[@"path"] isKindOfClass:[NSString class]]) return 2;
            recognize.recognitionLanguages = @[@"zh-Hans", @"en-US"];
            recognize.usesLanguageCorrection = NO;
            VNImageRequestHandler *handler = [[VNImageRequestHandler alloc] initWithURL:[NSURL fileURLWithPath:request[@"path"]] options:@{}];
            if (![handler performRequests:@[recognize] error:&error]) return 3;
            NSMutableArray *lines = [NSMutableArray array];
            for (VNRecognizedTextObservation *observation in recognize.results) {
                VNRecognizedText *candidate = [[observation topCandidates:1] firstObject];
                if (candidate.string.length) [lines addObject:candidate.string];
            }
            result[@"text"] = [lines componentsJoinedByString:@"\n"];
        }
        NSData *output = [NSJSONSerialization dataWithJSONObject:result options:0 error:nil];
        if (!output) return 4;
        [[NSFileHandle fileHandleWithStandardOutput] writeData:output];
    }
    return 0;
}
