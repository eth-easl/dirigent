//
// Created by lcvetkovic on 5/16/23.
//

#include "gtest/gtest.h"
#include "ingress/request_buffer.h"

TEST(RequestBufferTestSuite, TestOperations) {
    ingress::RequestBuffer<int> buffer;

    EXPECT_EQ(buffer.Length(), 0);

    buffer.Enqueue(1);
    EXPECT_EQ(buffer.Length(), 1);

    buffer.Enqueue(2);
    EXPECT_EQ(buffer.Length(), 2);

    buffer.Enqueue(3);
    EXPECT_EQ(buffer.Length(), 3);

    EXPECT_EQ(buffer.Empty(), false);

    EXPECT_EQ(buffer.Dequeue(), 1);
    EXPECT_EQ(buffer.Length(), 2);

    EXPECT_EQ(buffer.Dequeue(), 2);
    EXPECT_EQ(buffer.Length(), 1);

    EXPECT_EQ(buffer.Dequeue(), 3);
    EXPECT_EQ(buffer.Length(), 0);

    EXPECT_EQ(buffer.Empty(), true);
}