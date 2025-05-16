<template>
    <div class="container mt-5">
        <h2>แบบสอบถาม DISC Model</h2>
        <div v-if="loading" class="loading-spinner">
            <!-- You can use a spinner component or a simple message -->
            <div class="spinner-border" role="status">
                <span class="visually-hidden">Loading...</span>
            </div>
        </div>
        <form v-else @submit.prevent="handleSubmit">
            <div v-for="(question, index) in questions" :key="index" class="mb-3">
                <label :for="'question-' + index" class="form-label">{{ question.text }}</label>
                <div v-for="(option, idx) in question.options" :key="idx" class="form-check">
                    <input :id="'question-' + index + '-option-' + idx" type="radio" v-model="responses[index]"
                        :value="option" class="form-check-input">
                    <label :for="'question-' + index + '-option-' + idx" class="form-check-label">{{ option }}</label>
                </div>
            </div>
            <button type="submit" class="btn btn-primary">Submit</button>
        </form>
    </div>
</template>
<script>
import axios from 'axios';
import liff from "@line/liff";
export default {
    data() {
        return {
            loading: true,
            idToken: null,
            profile: null,
            context: null,
            groupId: null,
            questions: [
                {
                    text: "1. เมื่อทำงานในกลุ่ม คุณมักจะ...",
                    options: [
                        "A. เป็นผู้นำและกำหนดทิศทาง",
                        "B. สร้างบรรยากาศให้ทีมรู้สึกดี",
                        "C. ทำงานร่วมกับคนอื่นอย่างราบรื่น",
                        "D. ตรวจสอบรายละเอียดและความถูกต้อง"
                    ]
                },
                {
                    text: "2. เมื่อเจอสถานการณ์ใหม่ที่ไม่เคยเจอมาก่อน คุณจะ...",
                    options: [
                        "A. ลุยทันทีไม่รอใคร",
                        "B. อยากรู้จักคนอื่นและพูดคุย",
                        "C. ขอคำแนะนำจากคนรอบตัวก่อน",
                        "D. หาข้อมูล วิเคราะห์ ก่อนตัดสินใจ"
                    ]
                },
                {
                    text: "3. คุณรู้สึกภูมิใจที่สุดเมื่อ...",
                    options: [
                        "A. บรรลุเป้าหมายหรือความสำเร็จ",
                        "B. ทุกคนในทีมรู้สึกสนุกและพอใจ",
                        "C. งานราบรื่นโดยไม่มีปัญหา",
                        "D. งานมีความถูกต้องและมีคุณภาพสูง"
                    ]
                },
                {
                    text: "4. เมื่อต้องทำงานภายใต้แรงกดดัน คุณมักจะ...",
                    options: [
                        "A. เร่งผลักดันทีมให้เดินหน้า",
                        "B. ใช้พลังบวกปลุกใจทีม",
                        "C. ค่อยๆ ประสานงานและแก้ไขปัญหา",
                        "D. วางแผนอย่างรอบคอบและทำตามลำดับขั้น"
                    ]
                },
                {
                    text: "5. ถ้าให้เลือกสิ่งที่คุณให้ความสำคัญที่สุดในการทำงาน...",
                    options: [
                        "A. ประสิทธิภาพและความสำเร็จ",
                        "B. ความสัมพันธ์กับเพื่อนร่วมงาน",
                        "C. ความมั่นคงและความสม่ำเสมอ",
                        "D. ความถูกต้องและความเป็นระบบ"
                    ]
                }
            ]
            ,
            responses: [
                // "B. สร้างบรรยากาศให้ทีมรู้สึกดี",
                // "B. อยากรู้จักคนอื่นและพูดคุย",
                // "B. ทุกคนในทีมรู้สึกสนุกและพอใจ",
                // "B. ใช้พลังบวกปลุกใจทีม",
                // "B. ความสัมพันธ์กับเพื่อนร่วมงาน",
            ]
        };
    },
    beforeCreate() {
        liff
            .init({
                liffId: '2006952659-7320eNlX'
            })
            .then(() => {
                this.message = "LIFF init succeeded.";
            })
            .catch((e) => {
                this.message = "LIFF init failed.";
                this.error = `${e}`;
            });
    },
    async mounted() {
        await this.checkLiffLogin()
    },
    methods: {
        async checkLiffLogin() {
            await liff.ready.then(async () => {
                if (!liff.isLoggedIn()) {
                    liff.login({ redirectUri: window.location })
                } else {

                    this.idToken = await liff.getIDToken();
                    this.context = await liff.getContext();
                    this.groupId = this.$route.query.groupId
                    console.log(this.context.type);

                    this.loading = false

                }
            })
        },
        async handleSubmit() {
            this.loading = true
            if (this.hasUnansweredQuestions()) {
                alert("กรุณาตอบคำถามให้ครบทุกข้อ");
                this.loading = false
                return;
            }


            const answerData = {
                answers: Array.from(this.responses),
            };
            try {
                const response = await axios.post(`https://19c6236faadc.ngrok.app/submit-answer`,
                    answerData,
                    {
                        headers: {
                            Authorization: `${this.idToken}`,
                            GroupId: this.groupId,
                        },
                    }
                );
                if (liff.isInClient()) {
                    let message = `ฉันได้ประเมินเรียบร้อยแล้ว`
                    if (this.context.type === "utou") {
                        message = `ฉันได้ประเมินเรียบร้อยแล้ว ฉันได้กลุ่ม ${response.data.data.model}`
                    }

                    await liff.sendMessages([
                        {
                            type: "text",
                            text: message,
                        },
                    ]).then(() => {
                        liff.closeWindow()
                        this.loading = false

                    }).catch((err) => {
                        console.log("error", err);
                    });
                } else {

                    alert(`ผลที่ได้คือคุณเป็นกลุ่ม: ${response.data.data.model}`)
                    this.loading = false

                }

            } catch (error) {
                this.loading = false
                throw new Error(`handleSubmit: ${error}`);
            }

            // ทำการประมวลผลข้อมูลที่ได้จากแบบสอบถาม
        },
        hasUnansweredQuestions() {
            return this.responses.length < this.questions.length || this.responses.includes(undefined);
        }



    },
};
</script>

<style>
.error {
    color: red;
}
</style>